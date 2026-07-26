package model

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

var (
	ErrTaskLogBackfillInProgress = errors.New("task log summary backfill is already in progress")
	taskLogBackfillMutex         sync.Mutex
)

type TaskLogBackfillResult struct {
	ScannedTasks          int `json:"scanned_tasks"`
	MatchedTasks          int `json:"matched_tasks"`
	SummaryLogsUpdated    int `json:"summary_logs_updated"`
	AdjustmentLogsUpdated int `json:"adjustment_logs_updated"`
	TokenLogsUpdated      int `json:"token_logs_updated"`
	TaskLinksUpdated      int `json:"task_links_updated"`
}

type taskLogBackfillEntry struct {
	log   *Log
	other map[string]interface{}
}

// BackfillTaskLogSummaries folds historical async-task settlement/refund rows
// into their submit-time consume row. Quota columns remain untouched so
// accounting and profit history keep their original meaning.
func BackfillTaskLogSummaries() (TaskLogBackfillResult, error) {
	result := TaskLogBackfillResult{}
	if !taskLogBackfillMutex.TryLock() {
		return result, ErrTaskLogBackfillInProgress
	}
	defer taskLogBackfillMutex.Unlock()
	if DB == nil || LOG_DB == nil || common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return result, nil
	}

	const batchSize = 200
	lastID := int64(0)
	for {
		var tasks []*Task
		if err := DB.Where("id > ?", lastID).
			Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
			Order("id ASC").Limit(batchSize).Find(&tasks).Error; err != nil {
			return result, err
		}
		if len(tasks) == 0 {
			return result, nil
		}

		for _, task := range tasks {
			lastID = task.ID
			result.ScannedTasks++
			if task.PrivateData.ConsumeLogId > 0 {
				continue
			}
			entries, err := findHistoricalTaskLogs(task)
			if err != nil {
				return result, err
			}
			initialIndex := historicalTaskSubmitLogIndex(entries)
			if initialIndex < 0 {
				continue
			}
			result.MatchedTasks++

			initial := entries[initialIndex]
			preConsumed := initial.log.Quota
			actualQuota := task.Quota
			actualFound := false
			hasRefund := false
			reason := ""
			noRefund := false
			for i, entry := range entries {
				if i == initialIndex {
					continue
				}
				if value, ok := taskLogJSONInt(entry.other["pre_consumed_quota"]); ok {
					preConsumed = value
				}
				if value, ok := taskLogJSONInt(entry.other["actual_quota"]); ok {
					actualQuota = value
					actualFound = true
				}
				if entry.log.Type == LogTypeRefund {
					hasRefund = true
				}
				if entry.other["no_refund"] == true {
					noRefund = true
				}
				if value, ok := entry.other["reason"].(string); ok && value != "" {
					reason = value
				} else if entry.log.Content != "" {
					reason = entry.log.Content
				}
			}

			stage := "settle"
			if task.Status == TaskStatusFailure {
				if (actualFound && actualQuota == 0) || (!actualFound && hasRefund) {
					stage = "refund"
					actualQuota = 0
				} else if !actualFound && !hasRefund {
					noRefund = true
				}
				if reason == "" {
					reason = task.FailReason
				}
			} else if !actualFound && historicalTaskIsPerCall(task, initial) {
				stage = "final"
			}

			billedUsage := historicalTaskBilledUsage(task, entries)
			summaryPatch := map[string]interface{}{
				"task_summary":       true,
				"task_billing_stage": stage,
				"pre_consumed_quota": preConsumed,
				"actual_quota":       actualQuota,
			}
			if billedUsage > 0 {
				summaryPatch["billed_usage"] = billedUsage
			}
			if reason != "" {
				summaryPatch["reason"] = reason
			}
			if noRefund {
				summaryPatch["no_refund"] = true
			}
			summaryChanged, tokenChanged, err := updateHistoricalTaskLog(
				initial, summaryPatch, billedUsage, billedUsage > 0,
			)
			if err != nil {
				return result, err
			}
			if summaryChanged {
				result.SummaryLogsUpdated++
			}
			if tokenChanged {
				result.TokenLogsUpdated++
			}

			adjustmentPatch := map[string]interface{}{
				"task_adjust":        true,
				"task_billing_stage": stage,
				"pre_consumed_quota": preConsumed,
				"actual_quota":       actualQuota,
			}
			if reason != "" {
				adjustmentPatch["reason"] = reason
			}
			if noRefund {
				adjustmentPatch["no_refund"] = true
			}
			for i, entry := range entries {
				if i == initialIndex {
					continue
				}
				changed, _, err := updateHistoricalTaskLog(entry, adjustmentPatch, 0, entry.log.CompletionTokens > 0)
				if err != nil {
					return result, err
				}
				if changed {
					result.AdjustmentLogsUpdated++
				}
			}

			task.PrivateData.ConsumeLogId = initial.log.Id
			if err := DB.Model(&Task{}).Where("id = ?", task.ID).
				Update("private_data", task.PrivateData).Error; err != nil {
				return result, err
			}
			result.TaskLinksUpdated++
		}
	}
}

func findHistoricalTaskLogs(task *Task) ([]taskLogBackfillEntry, error) {
	condition, pattern, err := buildLogLikeCondition("other", "%"+task.TaskID+"%")
	if err != nil {
		return nil, err
	}
	var logs []*Log
	if err := LOG_DB.Where("user_id = ?", task.UserId).
		Where(condition, pattern).Order("created_at ASC, id ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	entries := make([]taskLogBackfillEntry, 0, len(logs))
	for _, log := range logs {
		other := map[string]interface{}{}
		if common.UnmarshalJsonStr(log.Other, &other) != nil || other["is_task"] != true {
			continue
		}
		if taskID, ok := other["task_id"].(string); !ok || taskID != task.TaskID {
			continue
		}
		entries = append(entries, taskLogBackfillEntry{log: log, other: other})
	}
	return entries, nil
}

func historicalTaskSubmitLogIndex(entries []taskLogBackfillEntry) int {
	fallback := -1
	for i, entry := range entries {
		if entry.log.Type != LogTypeConsume {
			continue
		}
		if fallback < 0 {
			fallback = i
		}
		if requestPath, ok := entry.other["request_path"].(string); ok && requestPath != "" {
			return i
		}
	}
	return fallback
}

func historicalTaskIsPerCall(task *Task, initial taskLogBackfillEntry) bool {
	if task.PrivateData.BillingContext != nil && task.PrivateData.BillingContext.PerCallBilling {
		return true
	}
	if modelPrice, ok := taskLogJSONNumber(initial.other["model_price"]); ok && modelPrice >= 0 {
		return true
	}
	return strings.Contains(initial.log.Content, "按次计费")
}

func historicalTaskBilledUsage(task *Task, entries []taskLogBackfillEntry) int {
	root := map[string]interface{}{}
	if len(task.Data) > 0 && common.Unmarshal(task.Data, &root) == nil {
		paths := [][]string{
			{"task", "usage", "total_tokens"},
			{"usage", "total_tokens"},
			{"data", "usage", "total_tokens"},
			{"task", "usage", "completion_tokens"},
			{"usage", "completion_tokens"},
			{"data", "usage", "completion_tokens"},
		}
		for _, path := range paths {
			value := interface{}(root)
			for _, key := range path {
				object, ok := value.(map[string]interface{})
				if !ok {
					value = nil
					break
				}
				value = object[key]
			}
			if usage, ok := taskLogJSONInt(value); ok && usage > 0 {
				return usage
			}
		}
	}
	for _, entry := range entries {
		if usage, ok := taskLogJSONInt(entry.other["billed_usage"]); ok && usage > 0 {
			return usage
		}
		if entry.log.CompletionTokens > 0 {
			return entry.log.CompletionTokens
		}
	}
	return 0
}

func taskLogJSONNumber(value interface{}) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, !math.IsNaN(number) && !math.IsInf(number, 0)
	case float32:
		value := float64(number)
		return value, !math.IsNaN(value) && !math.IsInf(value, 0)
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
	default:
		return 0, false
	}
}

func taskLogJSONInt(value interface{}) (int, bool) {
	number, ok := taskLogJSONNumber(value)
	if !ok || number < math.MinInt32 || number > math.MaxInt32 || math.Trunc(number) != number {
		return 0, false
	}
	return int(number), true
}

func updateHistoricalTaskLog(entry taskLogBackfillEntry, patch map[string]interface{}, completionTokens int, updateTokens bool) (bool, bool, error) {
	before := common.MapToJsonStr(entry.other)
	for key, value := range patch {
		entry.other[key] = value
	}
	after := common.MapToJsonStr(entry.other)
	otherChanged := before != after
	tokenChanged := updateTokens && entry.log.CompletionTokens != completionTokens
	if !otherChanged && !tokenChanged {
		return false, false, nil
	}
	updates := map[string]interface{}{}
	if otherChanged {
		updates["other"] = after
	}
	if tokenChanged {
		updates["completion_tokens"] = completionTokens
	}
	if err := LOG_DB.Model(&Log{}).Where("id = ?", entry.log.Id).Updates(updates).Error; err != nil {
		return false, false, err
	}
	entry.log.Other = after
	entry.log.CompletionTokens = completionTokens
	return true, tokenChanged, nil
}
