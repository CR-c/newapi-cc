package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetTopUpQuotaSnapshotUsesExactAmountBonus(t *testing.T) {
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalBonus := operation_setting.GetPaymentSetting().AmountBonus
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
		operation_setting.GetPaymentSetting().AmountBonus = originalBonus
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountBonus = map[int]int{100: 10}

	topUp := &model.TopUp{}
	require.NoError(t, setTopUpQuotaSnapshot(topUp, 100, 100*int(common.QuotaPerUnit)))
	assert.Equal(t, 100*int(common.QuotaPerUnit), topUp.PrincipalQuota)
	assert.Equal(t, 10*int(common.QuotaPerUnit), topUp.PromoQuota)

	withoutPreset := &model.TopUp{}
	require.NoError(t, setTopUpQuotaSnapshot(withoutPreset, 99, 99*int(common.QuotaPerUnit)))
	assert.Zero(t, withoutPreset.PromoQuota)
}

func TestTopUpPrincipalQuotaUsesRequestedDisplayAmount(t *testing.T) {
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	quota, err := topUpPrincipalQuotaFromRequestAmount(10)
	require.NoError(t, err)
	assert.Equal(t, 10*int(common.QuotaPerUnit), quota)

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeTokens
	quota, err = topUpPrincipalQuotaFromRequestAmount(12345)
	require.NoError(t, err)
	assert.Equal(t, 12345, quota)
}

func TestSetTopUpQuotaSnapshotTreatsTokenBonusAsQuota(t *testing.T) {
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalBonus := operation_setting.GetPaymentSetting().AmountBonus
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
		operation_setting.GetPaymentSetting().AmountBonus = originalBonus
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeTokens
	operation_setting.GetPaymentSetting().AmountBonus = map[int]int{1000: 100}

	topUp := &model.TopUp{}
	require.NoError(t, setTopUpQuotaSnapshot(topUp, 1000, 1000))
	assert.Equal(t, 1000, topUp.PrincipalQuota)
	assert.Equal(t, 100, topUp.PromoQuota)
}

func TestTopUpQuotaSnapshotRejectsOverflowingBonus(t *testing.T) {
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalBonus := operation_setting.GetPaymentSetting().AmountBonus
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
		operation_setting.GetPaymentSetting().AmountBonus = originalBonus
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountBonus = map[int]int{1: common.MaxQuota}

	err := setTopUpQuotaSnapshot(&model.TopUp{}, 1, 1)
	require.Error(t, err)
}

func TestGetCreemProductsWithBonusAddsPromoQuota(t *testing.T) {
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalBonus := operation_setting.GetPaymentSetting().AmountBonus
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
		operation_setting.GetPaymentSetting().AmountBonus = originalBonus
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountBonus = map[int]int{10: 2}
	raw := `[{"productId":"prod_test","name":"Test","price":10,"currency":"USD","quota":5000000}]`

	var products []CreemProduct
	require.NoError(t, common.UnmarshalJsonStr(getCreemProductsWithBonus(raw), &products))
	require.Len(t, products, 1)
	assert.Equal(t, int64(2*common.QuotaPerUnit), products[0].BonusQuota)
}

func TestCreemProductExplicitBonusOverridesAmountPreset(t *testing.T) {
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalBonus := operation_setting.GetPaymentSetting().AmountBonus
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
		operation_setting.GetPaymentSetting().AmountBonus = originalBonus
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountBonus = map[int]int{10: 2}
	product := &CreemProduct{Quota: 5_000_000, BonusQuota: 123}
	topUp := &model.TopUp{}

	require.NoError(t, setCreemTopUpQuotaSnapshot(topUp, product))
	assert.Equal(t, 5_000_000, topUp.PrincipalQuota)
	assert.Equal(t, 123, topUp.PromoQuota)
}
