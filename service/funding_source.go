package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// ---------------------------------------------------------------------------
// FundingSource — 资金来源接口（钱包 or 订阅）
// ---------------------------------------------------------------------------

// FundingSource 抽象了预扣费的资金来源。
type FundingSource interface {
	// Source 返回资金来源标识："wallet" 或 "subscription"
	Source() string
	// PreConsume 从该资金来源预扣 amount 额度
	PreConsume(amount int) error
	// Settle 根据差额调整资金来源（正数补扣，负数退还）
	Settle(delta int) error
	// Refund 退还所有预扣费
	Refund() error
}

// ---------------------------------------------------------------------------
// WalletFunding — 钱包资金来源实现
// ---------------------------------------------------------------------------

type WalletFunding struct {
	userId      int
	requestId   string
	allocation  model.WalletAllocation
	mutationSeq int
}

func (w *WalletFunding) Source() string { return BillingSourceWallet }

func (w *WalletFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	allocation, err := model.DebitWallet(w.userId, amount, "billing:"+w.requestId+":pre")
	if err != nil {
		return err
	}
	w.allocation = allocation
	return nil
}

func (w *WalletFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		eventKey := fmt.Sprintf("billing:%s:debit:%d", w.requestId, w.mutationSeq+1)
		allocation, err := model.DebitWallet(w.userId, delta, eventKey)
		if err != nil {
			return err
		}
		w.mutationSeq++
		w.allocation.PaidQuota += allocation.PaidQuota
		w.allocation.PromoQuota += allocation.PromoQuota
		w.allocation.LegacyQuota += allocation.LegacyQuota
		return nil
	}
	refund := takeWalletRefund(&w.allocation, -delta)
	if refund.Total() != -delta {
		return fmt.Errorf("wallet settlement refund exceeds consumed allocation")
	}
	eventKey := fmt.Sprintf("billing:%s:refund:%d", w.requestId, w.mutationSeq+1)
	if _, err := model.RefundWallet(w.userId, refund, eventKey); err != nil {
		w.allocation.PaidQuota += refund.PaidQuota
		w.allocation.PromoQuota += refund.PromoQuota
		w.allocation.LegacyQuota += refund.LegacyQuota
		return err
	}
	w.mutationSeq++
	return nil
}

func (w *WalletFunding) Refund() error {
	if w.allocation.Total() <= 0 {
		return nil
	}
	_, err := model.RefundWallet(w.userId, w.allocation, "billing:"+w.requestId+":full-refund")
	return err
}

func (w *WalletFunding) Allocation() model.WalletAllocation { return w.allocation }

func takeWalletRefund(allocation *model.WalletAllocation, amount int) model.WalletAllocation {
	if allocation == nil || amount <= 0 {
		return model.WalletAllocation{}
	}
	remaining := amount
	refund := model.WalletAllocation{}
	refund.PaidQuota = min(remaining, allocation.PaidQuota)
	remaining -= refund.PaidQuota
	refund.LegacyQuota = min(remaining, allocation.LegacyQuota)
	remaining -= refund.LegacyQuota
	refund.PromoQuota = min(remaining, allocation.PromoQuota)
	allocation.PaidQuota -= refund.PaidQuota
	allocation.LegacyQuota -= refund.LegacyQuota
	allocation.PromoQuota -= refund.PromoQuota
	return refund
}

// ---------------------------------------------------------------------------
// SubscriptionFunding — 订阅资金来源实现
// ---------------------------------------------------------------------------

type SubscriptionFunding struct {
	requestId      string
	userId         int
	modelName      string
	amount         int64 // 预扣的订阅额度（subConsume）
	subscriptionId int
	preConsumed    int64
	allocation     model.WalletAllocation
	// 以下字段在 PreConsume 成功后填充，供 RelayInfo 同步使用
	AmountTotal     int64
	AmountUsedAfter int64
	PlanId          int
	PlanTitle       string
}

func (s *SubscriptionFunding) Source() string { return BillingSourceSubscription }

func (s *SubscriptionFunding) PreConsume(_ int) error {
	// amount 参数被忽略，使用内部 s.amount（已在构造时根据 preConsumedQuota 计算）
	res, err := model.PreConsumeUserSubscription(s.requestId, s.userId, s.modelName, 0, s.amount)
	if err != nil {
		return err
	}
	s.subscriptionId = res.UserSubscriptionId
	s.preConsumed = res.PreConsumed
	s.allocation = res.FundingAllocation
	s.AmountTotal = res.AmountTotal
	s.AmountUsedAfter = res.AmountUsedAfter
	// 获取订阅计划信息
	if planInfo, err := model.GetSubscriptionPlanInfoByUserSubscriptionId(res.UserSubscriptionId); err == nil && planInfo != nil {
		s.PlanId = planInfo.PlanId
		s.PlanTitle = planInfo.PlanTitle
	}
	return nil
}

func (s *SubscriptionFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	targetAmount := s.allocation.Total() + delta
	if targetAmount < 0 {
		return fmt.Errorf("subscription settlement refund exceeds consumed allocation")
	}
	target, err := model.GetSubscriptionFundingAllocation(s.subscriptionId, int64(targetAmount))
	if err != nil {
		return err
	}
	if err := model.PostConsumeUserSubscriptionDelta(s.subscriptionId, int64(delta)); err != nil {
		return err
	}
	s.allocation = target
	return nil
}

func (s *SubscriptionFunding) Allocation() model.WalletAllocation { return s.allocation }

func (s *SubscriptionFunding) Refund() error {
	if s.preConsumed <= 0 {
		return nil
	}
	return refundWithRetry(func() error {
		return model.RefundSubscriptionPreConsume(s.requestId)
	})
}

// refundWithRetry 尝试多次执行退款操作以提高成功率，只能用于基于事务的退款函数！！！！！！
// try to refund with retries, only for refund functions based on transactions!!!
func refundWithRetry(fn func() error) error {
	if fn == nil {
		return nil
	}
	const maxAttempts = 3
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i < maxAttempts-1 {
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
	}
	return lastErr
}
