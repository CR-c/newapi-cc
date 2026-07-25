package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func seedFlowQuotaData(t *testing.T, quotaData QuotaData) {
	t.Helper()
	require.NoError(t, DB.Create(&quotaData).Error)
}

func seedFlowLookupData(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Create(&Channel{Id: 1, Name: "east"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 2, Name: "west"}).Error)
	require.NoError(t, DB.Create(&Token{Id: 11, UserId: 1, Key: "sk-primary", Name: "primary"}).Error)
	require.NoError(t, DB.Create(&Token{Id: 22, UserId: 2, Key: "sk-backup", Name: "backup"}).Error)
	require.NoError(t, DB.Delete(&Token{Id: 11}).Error)
}

func TestGetFlowQuotaDataUsesQuotaDataRoleSpecificDimensions(t *testing.T) {
	truncateTables(t)
	seedFlowLookupData(t)

	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 1,
		CreatedAt: 1000,
		Count:     2,
		Quota:     100,
		TokenUsed: 40,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 1,
		CreatedAt: 1100,
		Count:     1,
		Quota:     50,
		TokenUsed: 20,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 2,
		CreatedAt: 1200,
		Count:     1,
		Quota:     25,
		TokenUsed: 10,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    2,
		Username:  "bob",
		NodeName:  "node-b",
		TokenID:   22,
		UseGroup:  "default",
		ModelName: "gpt-b",
		ChannelID: 1,
		CreatedAt: 1300,
		Count:     3,
		Quota:     70,
		TokenUsed: 30,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		ModelName: "legacy",
		CreatedAt: 1400,
		Count:     99,
		Quota:     999,
		TokenUsed: 999,
	})

	rootRows, err := GetFlowQuotaData(900, 2000, "", 0, common.RoleRootUser)
	require.NoError(t, err)
	require.Len(t, rootRows, 3)
	// Token 11 was soft-deleted, so its name is intentionally left empty for the
	// frontend to render a localized "deleted (id)" label instead.
	require.Equal(t, FlowQuotaData{
		UserID:      1,
		Username:    "alice",
		NodeName:    "node-a",
		TokenID:     11,
		TokenName:   "",
		UseGroup:    "vip",
		ChannelID:   1,
		ChannelName: "east",
		ModelName:   "gpt-a",
		TokenUsed:   60,
		Count:       3,
		Quota:       150,
	}, *rootRows[0])
	// A token that still exists resolves to its current name.
	require.Equal(t, 22, rootRows[1].TokenID)
	require.Equal(t, "backup", rootRows[1].TokenName)

	adminRows, err := GetFlowQuotaData(900, 2000, "alice", 0, common.RoleAdminUser)
	require.NoError(t, err)
	require.Len(t, adminRows, 2)
	require.Equal(t, 0, adminRows[0].TokenID)
	require.Empty(t, adminRows[0].TokenName)
	require.Empty(t, adminRows[0].NodeName)
	require.Equal(t, "alice", adminRows[0].Username)
	require.Equal(t, "vip", adminRows[0].UseGroup)
	require.Equal(t, "east", adminRows[0].ChannelName)
	require.Equal(t, 150, adminRows[0].Quota)

	selfRows, err := GetFlowQuotaData(900, 2000, "", 1, common.RoleCommonUser)
	require.NoError(t, err)
	require.Len(t, selfRows, 1)
	require.Empty(t, selfRows[0].Username)
	require.Equal(t, 0, selfRows[0].ChannelID)
	require.Empty(t, selfRows[0].ChannelName)
	require.Empty(t, selfRows[0].TokenName)
	require.Equal(t, "vip", selfRows[0].UseGroup)
	require.Equal(t, 175, selfRows[0].Quota)
}

func TestLogQuotaDataSplitsRowsByUseGroupTokenChannelAndNode(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	LogQuotaData(QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 3661,
		UseGroup:  "vip",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "node-a",
		Quota:     100,
		TokenUsed: 40,
		Count:     1,
	})
	LogQuotaData(QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 3700,
		UseGroup:  "vip",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "node-a",
		Quota:     50,
		TokenUsed: 20,
		Count:     1,
	})
	LogQuotaData(QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 3700,
		UseGroup:  "default",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "node-a",
		Quota:     25,
		TokenUsed: 10,
		Count:     1,
	})

	SaveQuotaDataCache()

	var rows []QuotaData
	require.NoError(t, DB.Order("quota DESC").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, int64(3600), rows[0].CreatedAt)
	require.Equal(t, "vip", rows[0].UseGroup)
	require.Equal(t, 11, rows[0].TokenID)
	require.Equal(t, 1, rows[0].ChannelID)
	require.Equal(t, "node-a", rows[0].NodeName)
	require.Equal(t, 2, rows[0].Count)
	require.Equal(t, 150, rows[0].Quota)
	require.Equal(t, 60, rows[0].TokenUsed)
	require.Equal(t, "default", rows[1].UseGroup)
	require.Equal(t, 25, rows[1].Quota)
}

func TestRecordTaskBillingLogRefundDeductsDashboardQuota(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	prevExport := common.DataExportEnabled
	common.DataExportEnabled = true
	t.Cleanup(func() {
		common.DataExportEnabled = prevExport
		CacheQuotaDataLock.Lock()
		CacheQuotaData = make(map[string]*QuotaData)
		CacheQuotaDataLock.Unlock()
	})

	require.NoError(t, DB.Create(&User{Id: 42, Username: "refund-user", Role: common.RoleCommonUser}).Error)

	// Pre-consume style consume log (task submit).
	RecordTaskBillingLog(RecordTaskBillingLogParams{
		UserId:    42,
		LogType:   LogTypeConsume,
		ChannelId: 7,
		ModelName: "sora-2",
		Quota:     1000,
		TokenId:   0,
		Group:     "default",
		NodeName:  "node-a",
	})
	// Full failure refund (video / MJ / async task failure).
	RecordTaskBillingLog(RecordTaskBillingLogParams{
		UserId:    42,
		LogType:   LogTypeRefund,
		ChannelId: 7,
		ModelName: "sora-2",
		Quota:     1000,
		TokenId:   0,
		Group:     "default",
		NodeName:  "node-a",
	})
	// Partial delta refund (actual charge lower than pre-consume).
	RecordTaskBillingLog(RecordTaskBillingLogParams{
		UserId:    42,
		LogType:   LogTypeConsume,
		ChannelId: 7,
		ModelName: "sora-2",
		Quota:     500,
		TokenId:   0,
		Group:     "default",
		NodeName:  "node-a",
	})
	RecordTaskBillingLog(RecordTaskBillingLogParams{
		UserId:    42,
		LogType:   LogTypeRefund,
		ChannelId: 7,
		ModelName: "sora-2",
		Quota:     200,
		TokenId:   0,
		Group:     "default",
		NodeName:  "node-a",
	})

	SaveQuotaDataCache()

	var rows []QuotaData
	require.NoError(t, DB.Where("username = ?", "refund-user").Find(&rows).Error)
	require.Len(t, rows, 1)
	// Net: +1000 -1000 +500 -200 = 300; request count only from consume events.
	require.Equal(t, 300, rows[0].Quota)
	require.Equal(t, 2, rows[0].Count)
	require.Equal(t, "sora-2", rows[0].ModelName)
	require.Equal(t, 7, rows[0].ChannelID)
	require.Equal(t, "node-a", rows[0].NodeName)
}

func TestRebuildQuotaDataFromLogsNetsRefunds(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	prevNode := common.NodeName
	common.NodeName = "rebuild-node"
	t.Cleanup(func() {
		common.NodeName = prevNode
		CacheQuotaDataLock.Lock()
		CacheQuotaData = make(map[string]*QuotaData)
		CacheQuotaDataLock.Unlock()
	})

	// Seed a stale dashboard row that should be wiped by rebuild.
	require.NoError(t, DB.Create(&QuotaData{
		UserID:    1,
		Username:  "alice",
		ModelName: "stale-model",
		CreatedAt: 3600,
		UseGroup:  "vip",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "rebuild-node",
		Count:     9,
		Quota:     99999,
		TokenUsed: 9,
	}).Error)

	// Consume + full refund in same hour, plus a later partial refund hour.
	seedBillingLog(t, Log{
		UserId: 1, Username: "alice", ModelName: "sora-2", CreatedAt: 3700,
		Type: LogTypeConsume, Quota: 1000, PromptTokens: 10, CompletionTokens: 20,
		TokenId: 11, ChannelId: 1, Group: "vip",
	})
	seedBillingLog(t, Log{
		UserId: 1, Username: "alice", ModelName: "sora-2", CreatedAt: 3800,
		Type: LogTypeRefund, Quota: 1000, TokenId: 11, ChannelId: 1, Group: "vip",
	})
	seedBillingLog(t, Log{
		UserId: 1, Username: "alice", ModelName: "sora-2", CreatedAt: 7300,
		Type: LogTypeConsume, Quota: 500, PromptTokens: 5, CompletionTokens: 5,
		TokenId: 11, ChannelId: 1, Group: "vip",
	})
	seedBillingLog(t, Log{
		UserId: 1, Username: "alice", ModelName: "sora-2", CreatedAt: 7400,
		Type: LogTypeRefund, Quota: 200, TokenId: 11, ChannelId: 1, Group: "vip",
	})
	// Non-billing logs must be ignored.
	seedBillingLog(t, Log{
		UserId: 1, Username: "alice", ModelName: "sora-2", CreatedAt: 7500,
		Type: LogTypeTopup, Quota: 12345, TokenId: 11, ChannelId: 1, Group: "vip",
	})

	result, err := RebuildQuotaDataFromLogs()
	require.NoError(t, err)
	require.Equal(t, 4, result.LogScanned)
	require.Equal(t, 2, result.Rows)
	require.Equal(t, 300, result.NetQuota) // 1000-1000+500-200
	require.Equal(t, 2, result.Count)
	require.Equal(t, 40, result.TokenUsed) // only consume tokens: 30 + 10

	var rows []QuotaData
	require.NoError(t, DB.Order("created_at ASC").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, int64(3600), rows[0].CreatedAt)
	require.Equal(t, 0, rows[0].Quota)
	require.Equal(t, 1, rows[0].Count)
	require.Equal(t, 30, rows[0].TokenUsed)
	require.Equal(t, "rebuild-node", rows[0].NodeName)
	require.Equal(t, int64(7200), rows[1].CreatedAt)
	require.Equal(t, 300, rows[1].Quota)
	require.Equal(t, 1, rows[1].Count)
	require.Equal(t, 10, rows[1].TokenUsed)

	// Stale cache entries must not survive rebuild.
	require.Empty(t, CacheQuotaData)
}

func seedBillingLog(t *testing.T, log Log) {
	t.Helper()
	require.NoError(t, LOG_DB.Create(&log).Error)
}
