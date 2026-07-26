package model

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBackfillTaskLogSummariesMergesHistoricalBillingRows(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	originalMainType := common.MainDatabaseType()
	originalLogType := common.LogDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Task{}, &Log{}))
	DB, LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.SetDatabaseTypes(originalMainType, originalLogType)
	})

	tasks := []*Task{
		{
			TaskID: "task_settled", Platform: constant.TaskPlatform("59"), UserId: 1,
			ChannelId: 4, Quota: 60, Status: TaskStatusSuccess, SubmitTime: 100, FinishTime: 120,
			Data: json.RawMessage(`{"task":{"usage":{"total_tokens":12,"completion_tokens":12}}}`),
		},
		{
			TaskID: "task_refunded", Platform: constant.TaskPlatform("60"), UserId: 1,
			ChannelId: 5, Quota: 200, Status: TaskStatusFailure, SubmitTime: 200, FinishTime: 220,
			FailReason: "upstream failed",
		},
		{
			TaskID: "task_final", Platform: constant.TaskPlatform("55"), UserId: 2,
			ChannelId: 6, Quota: 300, Status: TaskStatusSuccess, SubmitTime: 300, FinishTime: 320,
		},
		{
			TaskID: "task_retained", Platform: constant.TaskPlatform("59"), UserId: 3,
			ChannelId: 4, Quota: 400, Status: TaskStatusFailure, SubmitTime: 400, FinishTime: 420,
			FailReason: "upstream charged",
		},
		{
			TaskID: "task_active", Platform: constant.TaskPlatform("59"), UserId: 4,
			ChannelId: 4, Quota: 500, Status: TaskStatusInProgress, SubmitTime: 500,
		},
	}
	for _, task := range tasks {
		require.NoError(t, db.Create(task).Error)
	}

	logs := []*Log{
		{Id: 1, UserId: 1, CreatedAt: 100, Type: LogTypeConsume, Quota: 100, ChannelId: 4,
			Other: `{"is_task":true,"task_id":"task_settled","request_path":"/v1/videos","model_price":-1}`},
		{Id: 2, UserId: 1, CreatedAt: 120, Type: LogTypeRefund, Quota: 40, CompletionTokens: 12, ChannelId: 4,
			Content: "token settlement", Other: `{"is_task":true,"task_id":"task_settled","pre_consumed_quota":100,"actual_quota":60}`},
		{Id: 3, UserId: 1, CreatedAt: 200, Type: LogTypeConsume, Quota: 200, ChannelId: 5,
			Other: `{"is_task":true,"task_id":"task_refunded","request_path":"/v1/videos","model_price":4}`},
		{Id: 4, UserId: 1, CreatedAt: 220, Type: LogTypeRefund, Quota: 200, ChannelId: 5,
			Other: `{"is_task":true,"task_id":"task_refunded"}`},
		{Id: 5, UserId: 2, CreatedAt: 300, Type: LogTypeConsume, Quota: 300, ChannelId: 6,
			Content: "per-call billing", Other: `{"is_task":true,"task_id":"task_final","request_path":"/v1/videos","model_price":1.5}`},
		{Id: 6, UserId: 3, CreatedAt: 400, Type: LogTypeConsume, Quota: 400, ChannelId: 4,
			Other: `{"is_task":true,"task_id":"task_retained","request_path":"/v1/videos","model_price":-1}`},
		{Id: 7, UserId: 4, CreatedAt: 500, Type: LogTypeConsume, Quota: 500, ChannelId: 4,
			Other: `{"is_task":true,"task_id":"task_active","request_path":"/v1/videos","model_price":-1}`},
	}
	for _, log := range logs {
		require.NoError(t, db.Create(log).Error)
	}

	result, err := BackfillTaskLogSummaries()
	require.NoError(t, err)
	assert.Equal(t, 4, result.MatchedTasks)
	assert.Equal(t, 4, result.SummaryLogsUpdated)
	assert.Equal(t, 2, result.AdjustmentLogsUpdated)
	assert.Equal(t, 1, result.TokenLogsUpdated)
	assert.Equal(t, 4, result.TaskLinksUpdated)

	var settled Log
	require.NoError(t, db.First(&settled, 1).Error)
	settledOther := map[string]interface{}{}
	require.NoError(t, common.UnmarshalJsonStr(settled.Other, &settledOther))
	assert.Equal(t, "settle", settledOther["task_billing_stage"])
	assert.Equal(t, float64(100), settledOther["pre_consumed_quota"])
	assert.Equal(t, float64(60), settledOther["actual_quota"])
	assert.Equal(t, float64(12), settledOther["billed_usage"])
	assert.Equal(t, true, settledOther["task_summary"])
	assert.Equal(t, 12, settled.CompletionTokens)

	var adjustment Log
	require.NoError(t, db.First(&adjustment, 2).Error)
	adjustmentOther := map[string]interface{}{}
	require.NoError(t, common.UnmarshalJsonStr(adjustment.Other, &adjustmentOther))
	assert.Equal(t, true, adjustmentOther["task_adjust"])
	assert.Equal(t, "settle", adjustmentOther["task_billing_stage"])
	assert.Zero(t, adjustment.CompletionTokens)

	var refunded Log
	require.NoError(t, db.First(&refunded, 3).Error)
	refundedOther := map[string]interface{}{}
	require.NoError(t, common.UnmarshalJsonStr(refunded.Other, &refundedOther))
	assert.Equal(t, "refund", refundedOther["task_billing_stage"])
	assert.Equal(t, float64(200), refundedOther["pre_consumed_quota"])
	assert.Equal(t, float64(0), refundedOther["actual_quota"])

	var final Log
	require.NoError(t, db.First(&final, 5).Error)
	finalOther := map[string]interface{}{}
	require.NoError(t, common.UnmarshalJsonStr(final.Other, &finalOther))
	assert.Equal(t, "final", finalOther["task_billing_stage"])
	assert.Equal(t, float64(300), finalOther["actual_quota"])

	var retained Log
	require.NoError(t, db.First(&retained, 6).Error)
	retainedOther := map[string]interface{}{}
	require.NoError(t, common.UnmarshalJsonStr(retained.Other, &retainedOther))
	assert.Equal(t, "settle", retainedOther["task_billing_stage"])
	assert.Equal(t, true, retainedOther["no_refund"])
	assert.Equal(t, float64(400), retainedOther["actual_quota"])

	var active Log
	require.NoError(t, db.First(&active, 7).Error)
	assert.JSONEq(t, logs[6].Other, active.Other)

	for _, taskID := range []string{"task_settled", "task_refunded", "task_final", "task_retained"} {
		var task Task
		require.NoError(t, db.Where("task_id = ?", taskID).First(&task).Error)
		assert.Positive(t, task.PrivateData.ConsumeLogId)
	}

	second, err := BackfillTaskLogSummaries()
	require.NoError(t, err)
	assert.Zero(t, second.MatchedTasks)
	assert.Zero(t, second.SummaryLogsUpdated)
	assert.Zero(t, second.AdjustmentLogsUpdated)
}
