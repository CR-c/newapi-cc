package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrateSubscriptionFundingAttributionUsesConservativeSources(t *testing.T) {
	truncateTables(t)

	subs := []UserSubscription{
		{Id: 9701, UserId: 1, Source: "admin"},
		{Id: 9702, UserId: 2, Source: "order"},
		{Id: 9703, UserId: 3, Source: PaymentMethodBalance},
	}
	require.NoError(t, DB.Create(&subs).Error)
	require.NoError(t, migrateSubscriptionFundingAttribution())

	var migrated []UserSubscription
	require.NoError(t, DB.Order("id asc").Find(&migrated).Error)
	require.Len(t, migrated, 3)
	assert.Equal(t, WalletAllocation{PromoQuota: 1_000_000}, subscriptionFundingAllocation(&migrated[0], 1_000_000))
	assert.Equal(t, WalletAllocation{LegacyQuota: 1_000_000}, subscriptionFundingAllocation(&migrated[1], 1_000_000))
	assert.Equal(t, WalletAllocation{LegacyQuota: 1_000_000}, subscriptionFundingAllocation(&migrated[2], 1_000_000))
}

func TestCreateUserSubscriptionAttributesAdminAndFreePlansToPromo(t *testing.T) {
	truncateTables(t)

	paidPlan := &SubscriptionPlan{Id: 9710, Title: "Paid", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth, DurationValue: 1}
	freePlan := &SubscriptionPlan{Id: 9711, Title: "Free", PriceAmount: 0, DurationUnit: SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, DB.Create([]*SubscriptionPlan{paidPlan, freePlan}).Error)

	var adminSub, freeSub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		adminSub, err = CreateUserSubscriptionFromPlanTx(tx, 10, paidPlan, "admin")
		if err != nil {
			return err
		}
		freeSub, err = CreateUserSubscriptionFromPlanTx(tx, 11, freePlan, "order")
		return err
	}))

	assert.Equal(t, WalletAllocation{PromoQuota: 100}, subscriptionFundingAllocation(adminSub, 100))
	assert.Equal(t, WalletAllocation{PromoQuota: 100}, subscriptionFundingAllocation(freeSub, 100))
}

func TestBalanceSubscriptionInheritsWalletFundingAndPreConsumeRefundKeepsIt(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 100
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })

	user := &User{
		Id: 9720, Username: "mixed-subscription", Status: common.UserStatusEnabled,
		Quota: 100, PaidQuota: 50, PromoQuota: 30, LegacyUnknownQuota: 20,
		WalletVersion: CurrentWalletVersion,
	}
	plan := &SubscriptionPlan{
		Id: 9721, Title: "Mixed", PriceAmount: 1, Enabled: true,
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 1_000,
	}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, PurchaseSubscriptionWithBalance(user.Id, plan.Id))

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ?", user.Id).First(&sub).Error)
	assert.Equal(t, 50_000, sub.FundingPaidPPM)
	assert.Equal(t, 930_000, sub.FundingPromoPPM)
	assert.Equal(t, 20_000, sub.FundingLegacyPPM)

	result, err := PreConsumeUserSubscription("req-mixed-subscription", user.Id, "test-model", 0, 100)
	require.NoError(t, err)
	assert.Equal(t, WalletAllocation{PaidQuota: 5, PromoQuota: 93, LegacyQuota: 2}, result.FundingAllocation)
	require.NoError(t, RefundSubscriptionPreConsume("req-mixed-subscription"))

	var record SubscriptionPreConsumeRecord
	require.NoError(t, DB.Where("request_id = ?", "req-mixed-subscription").First(&record).Error)
	assert.Equal(t, "refunded", record.Status)
	assert.Equal(t, 5, record.PaidQuota)
	assert.Equal(t, 93, record.PromoQuota)
	assert.Equal(t, 2, record.LegacyQuota)
	require.NoError(t, DB.First(&sub, sub.Id).Error)
	assert.Zero(t, sub.AmountUsed)
}

func TestSubscriptionFundingCapsPaidRevenueAtPurchaseValue(t *testing.T) {
	truncateTables(t)
	plan := &SubscriptionPlan{
		Id: 9730, Title: "Discounted", PriceAmount: 1,
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1,
		TotalAmount: 200, QuotaResetPeriod: SubscriptionResetNever,
	}
	require.NoError(t, DB.Create(plan).Error)

	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, 20, plan, "order")
		if err != nil {
			return err
		}
		return setSubscriptionFundingFromPurchaseTx(tx, sub, plan, WalletAllocation{PaidQuota: 100})
	}))

	assert.Equal(t, 500_000, sub.FundingPaidPPM)
	assert.Equal(t, 500_000, sub.FundingPromoPPM)
	assert.Equal(t, WalletAllocation{PaidQuota: 100, PromoQuota: 100}, subscriptionFundingAllocation(sub, 200))
}

func TestResettingSubscriptionFundingIsConservativePromo(t *testing.T) {
	truncateTables(t)
	plan := &SubscriptionPlan{
		Id: 9740, Title: "Recurring quota", PriceAmount: 10,
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1,
		TotalAmount: 1_000, QuotaResetPeriod: SubscriptionResetDaily,
	}
	require.NoError(t, DB.Create(plan).Error)

	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, 21, plan, "order")
		if err != nil {
			return err
		}
		return setSubscriptionFundingFromPurchaseTx(tx, sub, plan, WalletAllocation{PaidQuota: 500})
	}))

	assert.Equal(t, WalletAllocation{PromoQuota: 1_000}, subscriptionFundingAllocation(sub, 1_000))
}

func TestCompleteSubscriptionOrderUsesVerifiedPaidAmount(t *testing.T) {
	truncateTables(t)
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 100
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })

	plan := &SubscriptionPlan{
		Id: 9750, Title: "Verified payment", PriceAmount: 1, Enabled: true,
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1,
		TotalAmount: 200, QuotaResetPeriod: SubscriptionResetNever,
	}
	order := &SubscriptionOrder{
		UserId: 22, PlanId: plan.Id, Money: 1, TradeNo: "verified-paid-amount",
		PaymentProvider: PaymentProviderStripe, Status: common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Create(order).Error)

	require.NoError(t, CompleteSubscriptionOrderWithPaidAmount(
		order.TradeNo, `{"amount_total":50}`, PaymentProviderStripe, "", 0.5,
	))

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ?", order.UserId).First(&sub).Error)
	assert.Equal(t, 250_000, sub.FundingPaidPPM)
	assert.Equal(t, 750_000, sub.FundingPromoPPM)
	assert.Equal(t, WalletAllocation{PaidQuota: 50, PromoQuota: 150}, subscriptionFundingAllocation(&sub, 200))
	require.NoError(t, DB.First(order, order.Id).Error)
	assert.Equal(t, 0.5, order.Money)
}
