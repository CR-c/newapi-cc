package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// LogTaskConsumption 记录任务消费日志和统计信息（仅记录，不涉及实际扣费）。
// 实际扣费已由 BillingSession（PreConsumeBilling + SettleBilling）完成。
// 返回消费日志行 id（未持久化时为 0），供任务结算阶段原地补写汇总信息。
func LogTaskConsumption(c *gin.Context, info *relaycommon.RelayInfo) int {
	tokenName := c.GetString("token_name")
	logContent := fmt.Sprintf("操作 %s", info.Action)
	// 支持任务仅按次计费
	if common.StringsContains(constant.TaskPricePatches, info.OriginModelName) {
		logContent = fmt.Sprintf("%s，按次计费", logContent)
	} else {
		if otherRatios := info.PriceData.OtherRatios(); len(otherRatios) > 0 {
			var contents []string
			for key, ra := range otherRatios {
				if 1.0 != ra {
					contents = append(contents, fmt.Sprintf("%s: %.2f", key, ra))
				}
			}
			if len(contents) > 0 {
				logContent = fmt.Sprintf("%s, 计算参数：%s", logContent, strings.Join(contents, ", "))
			}
		}
	}
	other := make(map[string]interface{})
	other["is_task"] = true
	other["media_type"] = "video"
	// 计费阶段标记：按次计费提交即终结，否则为预扣费、等待轮询结算
	if common.StringsContains(constant.TaskPricePatches, info.OriginModelName) || info.PriceData.UsePrice {
		other["task_billing_stage"] = "final"
	} else {
		other["task_billing_stage"] = "pre_consume"
	}
	if info.TaskRelayInfo != nil && info.PublicTaskID != "" {
		other["task_id"] = info.PublicTaskID
	}
	other["request_path"] = c.Request.URL.Path
	other["model_price"] = info.PriceData.ModelPrice
	if info.PriceData.ModelRatio > 0 {
		other["model_ratio"] = info.PriceData.ModelRatio
	}
	other["group_ratio"] = info.PriceData.GroupRatioInfo.GroupRatio
	if info.CostTier != "" {
		other["cost_tier"] = info.CostTier
		other["other_multiplier"] = info.PriceData.OtherRatioMultiplier()
	}
	if info.ProfitGeneration != nil {
		other["profit_generation"] = *info.ProfitGeneration
	}
	if info.PriceData.GroupRatioInfo.HasSpecialRatio {
		other["user_group_ratio"] = info.PriceData.GroupRatioInfo.GroupSpecialRatio
	}
	if info.IsModelMapped {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = info.UpstreamModelName
	}
	appendBillingInfo(info, other)
	attachQuotaSaturation(c, info, other)
	logId := model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
		ChannelId: info.ChannelId,
		ModelName: info.OriginModelName,
		TokenName: tokenName,
		Quota:     info.PriceData.Quota,
		Content:   logContent,
		TokenId:   info.TokenId,
		Group:     info.UsingGroup,
		Other:     other,
	})
	model.UpdateUserUsedQuotaAndRequestCount(info.UserId, info.PriceData.Quota)
	model.UpdateChannelUsedQuota(info.ChannelId, info.PriceData.Quota)
	return logId
}

// ---------------------------------------------------------------------------
// 异步任务计费辅助函数
// ---------------------------------------------------------------------------

// resolveTokenKey 通过 TokenId 运行时获取令牌 Key（用于 Redis 缓存操作）。
// 如果令牌已被删除或查询失败，返回空字符串。
func resolveTokenKey(ctx context.Context, tokenId int, taskID string) string {
	token, err := model.GetTokenById(tokenId)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("获取令牌 key 失败 (tokenId=%d, task=%s): %s", tokenId, taskID, err.Error()))
		return ""
	}
	return token.Key
}

// taskIsSubscription 判断任务是否通过订阅计费。
func taskIsSubscription(task *model.Task) bool {
	return task.PrivateData.BillingSource == BillingSourceSubscription && task.PrivateData.SubscriptionId > 0
}

// taskAdjustFunding 调整任务的资金来源（钱包或订阅），delta > 0 表示扣费，delta < 0 表示退还。
func taskAdjustFunding(task *model.Task, delta int) (model.WalletAllocation, error) {
	if taskIsSubscription(task) {
		current := task.PrivateData.WalletAllocation
		if current.Total() == 0 && task.Quota > 0 {
			var err error
			current, err = model.GetSubscriptionFundingAllocation(task.PrivateData.SubscriptionId, int64(task.Quota))
			if err != nil {
				return model.WalletAllocation{}, err
			}
		}
		targetQuota := task.Quota + delta
		if targetQuota < 0 {
			return model.WalletAllocation{}, fmt.Errorf("task subscription refund exceeds consumed allocation")
		}
		target, err := model.GetSubscriptionFundingAllocation(task.PrivateData.SubscriptionId, int64(targetQuota))
		if err != nil {
			return model.WalletAllocation{}, err
		}
		if err := model.PostConsumeUserSubscriptionDelta(task.PrivateData.SubscriptionId, int64(delta)); err != nil {
			return model.WalletAllocation{}, err
		}
		event := model.WalletAllocation{}
		if delta > 0 {
			event.PaidQuota = target.PaidQuota - current.PaidQuota
			event.PromoQuota = target.PromoQuota - current.PromoQuota
			event.LegacyQuota = target.LegacyQuota - current.LegacyQuota
		} else {
			event.PaidQuota = current.PaidQuota - target.PaidQuota
			event.PromoQuota = current.PromoQuota - target.PromoQuota
			event.LegacyQuota = current.LegacyQuota - target.LegacyQuota
		}
		task.PrivateData.WalletAllocation = target
		return event, nil
	}
	preV2Allocation := task.PrivateData.WalletAllocationVersion < model.CurrentWalletVersion
	if preV2Allocation {
		historicalTotal := task.PrivateData.WalletAllocation.Total()
		if historicalTotal > 0 {
			task.PrivateData.WalletAllocation = model.WalletAllocation{PaidQuota: historicalTotal}
		}
		task.PrivateData.WalletAllocationVersion = model.CurrentWalletVersion
	}
	if delta > 0 {
		allocation, err := model.DebitWallet(task.UserId, delta,
			fmt.Sprintf("task:%s:quota:%d:debit", task.TaskID, task.Quota+delta))
		if err != nil {
			return model.WalletAllocation{}, err
		}
		task.PrivateData.WalletAllocation.PaidQuota += allocation.PaidQuota
		task.PrivateData.WalletAllocation.PromoQuota += allocation.PromoQuota
		task.PrivateData.WalletAllocation.LegacyQuota += allocation.LegacyQuota
		return allocation, nil
	}
	consumedRefund := model.WalletAllocation{}
	if task.PrivateData.WalletAllocation.Total() == 0 {
		if preV2Allocation {
			consumedRefund.PaidQuota = -delta
		} else {
			consumedRefund.LegacyQuota = -delta
		}
	} else {
		consumedRefund = takeWalletRefund(&task.PrivateData.WalletAllocation, -delta)
	}
	if consumedRefund.Total() != -delta {
		return model.WalletAllocation{}, fmt.Errorf("task wallet refund exceeds consumed allocation")
	}
	refund := consumedRefund
	_, err := model.RefundWallet(task.UserId, refund,
		fmt.Sprintf("task:%s:quota:%d:refund", task.TaskID, task.Quota+delta))
	if err != nil {
		task.PrivateData.WalletAllocation.PaidQuota += consumedRefund.PaidQuota
		task.PrivateData.WalletAllocation.PromoQuota += consumedRefund.PromoQuota
		task.PrivateData.WalletAllocation.LegacyQuota += consumedRefund.LegacyQuota
	}
	return refund, err
}

// taskAdjustTokenQuota 调整任务的令牌额度，delta > 0 表示扣费，delta < 0 表示退还。
// 需要通过 resolveTokenKey 运行时获取 key（不从 PrivateData 中读取）。
func taskAdjustTokenQuota(ctx context.Context, task *model.Task, delta int) {
	if task.PrivateData.TokenId <= 0 || delta == 0 {
		return
	}
	tokenKey := resolveTokenKey(ctx, task.PrivateData.TokenId, task.TaskID)
	if tokenKey == "" {
		return
	}
	var err error
	if delta > 0 {
		err = model.DecreaseTokenQuota(task.PrivateData.TokenId, tokenKey, delta)
	} else {
		err = model.IncreaseTokenQuota(task.PrivateData.TokenId, tokenKey, -delta)
	}
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("调整令牌额度失败 (delta=%d, task=%s): %s", delta, task.TaskID, err.Error()))
	}
}

// taskBillingOther 从 task 的 BillingContext 构建日志 Other 字段。
func taskBillingOther(task *model.Task, eventAllocation ...model.WalletAllocation) map[string]interface{} {
	other := map[string]interface{}{
		"is_task":            true,
		"media_type":         string(model.MediaTypeVideo),
		"task_id":            task.TaskID,
		"billing_source":     task.PrivateData.BillingSource,
		"user_role_snapshot": task.PrivateData.UserRoleSnapshot,
	}
	if task.PrivateData.IsAdminUsage {
		other["is_admin_usage"] = true
	}
	allocation := task.PrivateData.WalletAllocation
	if len(eventAllocation) > 0 {
		allocation = eventAllocation[0]
	}
	if allocation.PaidQuota != 0 || allocation.PromoQuota != 0 || allocation.LegacyQuota != 0 {
		other["wallet_funding"] = map[string]interface{}{
			"version":      1,
			"paid_quota":   allocation.PaidQuota,
			"promo_quota":  allocation.PromoQuota,
			"legacy_quota": allocation.LegacyQuota,
		}
	}
	if bc := task.PrivateData.BillingContext; bc != nil {
		other["model_price"] = bc.ModelPrice
		if bc.ModelRatio > 0 {
			other["model_ratio"] = bc.ModelRatio
		}
		other["group_ratio"] = bc.GroupRatio
		if bc.CostTier != "" {
			other["cost_tier"] = bc.CostTier
			otherMultiplier := bc.OtherMultiplier
			if otherMultiplier <= 0 {
				if priceData := taskBillingContextPriceData(bc); priceData != nil {
					otherMultiplier = priceData.OtherRatioMultiplier()
				}
			}
			if otherMultiplier > 0 {
				other["other_multiplier"] = otherMultiplier
			}
		}
		if bc.ProfitGeneration != nil {
			other["profit_generation"] = *bc.ProfitGeneration
		}
		if priceData := taskBillingContextPriceData(bc); priceData != nil {
			for k, v := range priceData.OtherRatios() {
				other[k] = v
			}
		}
	}
	props := task.Properties
	if props.UpstreamModelName != "" && props.UpstreamModelName != props.OriginModelName {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = props.UpstreamModelName
	}
	return other
}

func taskBillingContextPriceData(bc *model.TaskBillingContext) *types.PriceData {
	if bc == nil || len(bc.OtherRatios) == 0 {
		return nil
	}
	priceData := &types.PriceData{}
	if !priceData.ReplaceOtherRatios(bc.OtherRatios) {
		return nil
	}
	return priceData
}

// taskModelName 从 BillingContext 或 Properties 中获取模型名称。
func taskModelName(task *model.Task) string {
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OriginModelName != "" {
		return bc.OriginModelName
	}
	return task.Properties.OriginModelName
}

// ---------------------------------------------------------------------------
// 任务汇总日志行（单行演进）
//
// 提交时的消费日志行是任务在日志列表中的唯一展示行。结算/退款阶段通过
// finalizeTaskSummaryLog 把最终金额、上游用量、阶段标记补写进该行的 other
// JSON；差额调整行仍照常写入（供看板、利润分析、对账使用），但带上
// "task_adjust":true 标记后从日志列表中隐藏（见 model.hideTaskAdjustLogs）。
// ---------------------------------------------------------------------------

// taskSummaryPatch 构建任务汇总日志行的终态 other 补丁。
// stage 取值沿用日志前端已识别的枚举："settle"（已结算）或 "refund"（全额退款）。
func taskSummaryPatch(preConsumed, actualQuota, billedUsage int, reason string) map[string]interface{} {
	patch := map[string]interface{}{
		"task_summary":       true,
		"task_billing_stage": "settle",
		"pre_consumed_quota": preConsumed,
		"actual_quota":       actualQuota,
	}
	if billedUsage > 0 {
		patch["billed_usage"] = billedUsage
	}
	if reason != "" {
		patch["reason"] = reason
	}
	return patch
}

// finalizeTaskSummaryLog 将补丁和最终用量写到任务提交时的消费日志行上。
// 旧任务（无 ConsumeLogId）、日志未持久化或 ClickHouse 日志库时返回 false，
// 调用方应保持旧的多行展示（不给差额行打隐藏标记）。
func finalizeTaskSummaryLog(task *model.Task, patch map[string]interface{}, completionTokens int) bool {
	if task == nil || task.PrivateData.ConsumeLogId <= 0 {
		return false
	}
	return model.UpdateTaskLogSummary(task.PrivateData.ConsumeLogId, patch, completionTokens)
}

// KeepTaskChargeOnFailure 处理"上游已计费"的失败任务：有上游用量时先按
// 实际 token 结算；无法按 token 重算时保留预扣额度。两种情况都不执行失败
// 全额退款，并在汇总行记录原因。
func KeepTaskChargeOnFailure(ctx context.Context, task *model.Task, billedUsage int, reason string) {
	if task == nil || task.Quota == 0 {
		return
	}
	preConsumedQuota := task.Quota
	settledByUsage := billedUsage > 0 && RecalculateTaskQuotaByTokens(ctx, task, billedUsage)
	logger.LogInfo(ctx, fmt.Sprintf("任务 %s 失败但上游已计费，不退款（%s，%s）",
		task.TaskID, logger.LogQuota(task.Quota), reason))
	patch := map[string]interface{}{
		"no_refund": true,
		"reason":    reason,
	}
	if billedUsage > 0 {
		patch["billed_usage"] = billedUsage
	}
	if !settledByUsage {
		for key, value := range taskSummaryPatch(preConsumedQuota, task.Quota, billedUsage, reason) {
			patch[key] = value
		}
	}
	if finalizeTaskSummaryLog(task, patch, billedUsage) {
		return
	}
	// 旧任务没有汇总行可写，补一条零额消费日志留痕（Quota=0 不影响看板）。
	other := taskBillingOther(task)
	other["task_billing_stage"] = "settle"
	other["pre_consumed_quota"] = preConsumedQuota
	other["actual_quota"] = task.Quota
	other["no_refund"] = true
	other["reason"] = reason
	if billedUsage > 0 {
		other["billed_usage"] = billedUsage
	}
	auditTokens := max(billedUsage, 0)
	if settledByUsage {
		// token 重算的调整日志已经承载用量，旧任务的额外审计行只做不退款留痕。
		auditTokens = 0
	}
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:           task.UserId,
		LogType:          model.LogTypeConsume,
		Content:          "失败不退款（上游已计费）",
		ChannelId:        task.ChannelId,
		ModelName:        taskModelName(task),
		Quota:            0,
		CompletionTokens: auditTokens,
		TokenId:          task.PrivateData.TokenId,
		Group:            task.Group,
		Other:            other,
		NodeName:         task.PrivateData.NodeName,
	})
}

// RefundTaskQuota 统一的任务失败退款逻辑。
// 当异步任务失败时，将预扣的 quota 退还给用户（支持钱包和订阅），并退还令牌额度。
func RefundTaskQuota(ctx context.Context, task *model.Task, reason string) {
	quota := task.Quota
	if quota == 0 {
		return
	}

	// 1. 退还资金来源（钱包或订阅）
	allocation, err := taskAdjustFunding(task, -quota)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("退还资金来源失败 task %s: %s", task.TaskID, err.Error()))
		return
	}

	// 2. 退还令牌额度
	taskAdjustTokenQuota(ctx, task, -quota)
	model.UpdateUserUsedQuota(task.UserId, -quota)
	model.UpdateChannelUsedQuota(task.ChannelId, -quota)

	// 3. 汇总行终结为全额退款状态；成功后退款明细行从列表隐藏（仍写入供统计）
	summaryPatch := taskSummaryPatch(quota, 0, 0, reason)
	summaryPatch["task_billing_stage"] = "refund"
	summaryPatch["no_refund"] = false
	summaryFinalized := finalizeTaskSummaryLog(task, summaryPatch, 0)

	// 4. 记录退款日志
	other := taskBillingOther(task, allocation)
	other["reason"] = reason
	other["task_billing_stage"] = "refund"
	other["pre_consumed_quota"] = quota
	other["actual_quota"] = 0
	if summaryFinalized {
		other["task_adjust"] = true
	}
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    task.UserId,
		LogType:   model.LogTypeRefund,
		Content:   "",
		ChannelId: task.ChannelId,
		ModelName: taskModelName(task),
		Quota:     quota,
		TokenId:   task.PrivateData.TokenId,
		Group:     task.Group,
		Other:     other,
	})
}

// RecalculateTaskQuota 通用的异步差额结算。
// actualQuota 是任务完成后的实际应扣额度，与预扣额度 (task.Quota) 做差额结算。
// completionTokens 是上游返回的计费用量（token 或扣量单位），> 0 时随结算日志持久化。
// reason 用于日志记录（例如 "token重算" 或 "adaptor调整"）。
// clamps 可选：若计算 actualQuota 时发生额度饱和，将其记入日志 admin_info（仅管理员可见）。
func RecalculateTaskQuota(ctx context.Context, task *model.Task, actualQuota int, completionTokens int, reason string, clamps ...*common.QuotaClamp) bool {
	if actualQuota <= 0 {
		return false
	}
	preConsumedQuota := task.Quota
	quotaDelta := actualQuota - preConsumedQuota

	if quotaDelta == 0 {
		logger.LogInfo(ctx, fmt.Sprintf("任务 %s 预扣费准确（%s，%s）",
			task.TaskID, logger.LogQuota(actualQuota), reason))
		// 预扣费准确：把最终金额与上游用量补写到汇总日志行
		if finalizeTaskSummaryLog(task, taskSummaryPatch(preConsumedQuota, actualQuota, completionTokens, reason), completionTokens) {
			return true
		}
		if completionTokens <= 0 {
			return true
		}
		// 旧任务无汇总行时，写一条零额结算日志让上游用量对用户可见
		other := taskBillingOther(task, model.WalletAllocation{})
		other["task_billing_stage"] = "settle"
		other["pre_consumed_quota"] = preConsumedQuota
		other["actual_quota"] = actualQuota
		model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
			UserId:           task.UserId,
			LogType:          model.LogTypeConsume,
			Content:          "预扣费准确",
			ChannelId:        task.ChannelId,
			ModelName:        taskModelName(task),
			Quota:            0,
			CompletionTokens: completionTokens,
			TokenId:          task.PrivateData.TokenId,
			Group:            task.Group,
			Other:            other,
			NodeName:         task.PrivateData.NodeName,
		})
		return true
	}

	logger.LogInfo(ctx, fmt.Sprintf("任务 %s 差额结算：delta=%s（实际：%s，预扣：%s，%s）",
		task.TaskID,
		logger.LogQuota(quotaDelta),
		logger.LogQuota(actualQuota),
		logger.LogQuota(preConsumedQuota),
		reason,
	))

	// 调整资金来源
	allocation, err := taskAdjustFunding(task, quotaDelta)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("差额结算资金调整失败 task %s: %s", task.TaskID, err.Error()))
		return false
	}

	// 调整令牌额度
	taskAdjustTokenQuota(ctx, task, quotaDelta)

	task.Quota = actualQuota
	if err := task.UpdateQuota(); err != nil {
		logger.LogError(ctx, fmt.Sprintf("差额结算回写 quota 失败 task %s: %s", task.TaskID, err.Error()))
	}

	var logType int
	var logQuota int
	if quotaDelta > 0 {
		logType = model.LogTypeConsume
		logQuota = quotaDelta
	} else {
		logType = model.LogTypeRefund
		logQuota = -quotaDelta
	}
	model.UpdateUserUsedQuota(task.UserId, quotaDelta)
	model.UpdateChannelUsedQuota(task.ChannelId, quotaDelta)
	// 汇总行终结为已结算状态；成功后差额明细行从列表隐藏（仍写入供统计）
	summaryFinalized := finalizeTaskSummaryLog(task,
		taskSummaryPatch(preConsumedQuota, actualQuota, completionTokens, reason), completionTokens)

	other := taskBillingOther(task, allocation)
	other["task_billing_stage"] = "settle"
	other["pre_consumed_quota"] = preConsumedQuota
	other["actual_quota"] = actualQuota
	if summaryFinalized {
		other["task_adjust"] = true
	}
	for _, clamp := range clamps {
		attachQuotaSaturationToOther(other, clamp)
	}
	adjustmentTokens := max(completionTokens, 0)
	if summaryFinalized {
		adjustmentTokens = 0
	}
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:           task.UserId,
		LogType:          logType,
		Content:          reason,
		ChannelId:        task.ChannelId,
		ModelName:        taskModelName(task),
		Quota:            logQuota,
		CompletionTokens: adjustmentTokens,
		TokenId:          task.PrivateData.TokenId,
		Group:            task.Group,
		Other:            other,
		NodeName:         task.PrivateData.NodeName,
	})
	return true
}

// calculateTaskTokenQuota uses the immutable submit-time billing snapshot when
// available. Runtime settings are only a compatibility fallback for old tasks.
func calculateTaskTokenQuota(task *model.Task, totalTokens int) (int, *common.QuotaClamp, bool) {
	if task == nil || totalTokens <= 0 {
		return 0, nil, false
	}

	billingContext := task.PrivateData.BillingContext
	modelRatio := 0.0
	groupRatio := 0.0
	hasBillingSnapshot := billingContext != nil && billingContext.ModelRatio > 0
	if billingContext != nil {
		modelRatio = billingContext.ModelRatio
		groupRatio = billingContext.GroupRatio
	}
	if modelRatio <= 0 {
		var hasRatioSetting bool
		modelRatio, hasRatioSetting, _ = ratio_setting.GetModelRatio(taskModelName(task))
		if !hasRatioSetting || modelRatio <= 0 {
			return 0, nil, false
		}
	}
	if !hasBillingSnapshot {
		usingGroup := task.Group
		if usingGroup == "" {
			return 0, nil, false
		}
		groupRatio = ratio_setting.GetGroupRatio(usingGroup)
		if user, err := model.GetUserById(task.UserId, false); err == nil {
			if specialRatio, ok := ratio_setting.GetGroupGroupRatio(user.Group, usingGroup); ok {
				groupRatio = specialRatio
			}
		}
	}
	if groupRatio < 0 {
		return 0, nil, false
	}

	otherMultiplier := 1.0
	if priceData := taskBillingContextPriceData(billingContext); priceData != nil {
		otherMultiplier = priceData.OtherRatioMultiplier()
	}
	actualQuota, clamp := common.QuotaFromFloatChecked(float64(totalTokens) * modelRatio * groupRatio * otherMultiplier)
	if billingContext != nil && billingContext.CapSettlementAtPrecharge && actualQuota > task.Quota {
		actualQuota = task.Quota
	}
	return actualQuota, clamp, true
}

// RecalculateTaskQuotaByTokens 根据实际 token 消耗重新计费（异步差额结算）。
// 提交时计费快照优先于运行时配置，避免任务执行期间改价改变用户账单。
func RecalculateTaskQuotaByTokens(ctx context.Context, task *model.Task, totalTokens int) bool {
	actualQuota, clamp, ok := calculateTaskTokenQuota(task, totalTokens)
	if !ok {
		return false
	}
	reason := fmt.Sprintf("token重算：tokens=%d，按提交时计费快照结算", totalTokens)
	if task.PrivateData.BillingContext != nil && task.PrivateData.BillingContext.CapSettlementAtPrecharge && actualQuota == task.Quota {
		reason += "，不超过预扣额度"
	}
	return RecalculateTaskQuota(ctx, task, actualQuota, totalTokens, reason, clamp)
}
