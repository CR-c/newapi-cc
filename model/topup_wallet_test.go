package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopUpSetQuotaSnapshotValidatesAmounts(t *testing.T) {
	topUp := &TopUp{}

	require.NoError(t, topUp.SetQuotaSnapshot(500, 50))
	assert.Equal(t, 500, topUp.PrincipalQuota)
	assert.Equal(t, 50, topUp.PromoQuota)
	assert.Equal(t, CurrentTopUpSnapshotVersion, topUp.SnapshotVersion)

	require.Error(t, topUp.SetQuotaSnapshot(0, 0))
	require.Error(t, topUp.SetQuotaSnapshot(100, -1))
	require.Error(t, topUp.SetQuotaSnapshot(common.MaxQuota, 1))
}

func TestCompleteTopUpCreditsSnapshotBucketsExactlyOnce(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&QuotaLedger{}))
	tradeNo := fmt.Sprintf("snapshot-topup-%d", time.Now().UnixNano())
	user := &User{Username: tradeNo, Password: "password", Status: common.UserStatusEnabled, WalletVersion: CurrentWalletVersion}
	require.NoError(t, DB.Create(user).Error)
	topUp := &TopUp{
		UserId:          user.Id,
		Amount:          1,
		Money:           1,
		TradeNo:         tradeNo,
		PaymentMethod:   PaymentMethodWaffo,
		PaymentProvider: PaymentProviderWaffo,
		CreateTime:      common.GetTimestamp(),
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, topUp.SetQuotaSnapshot(500, 75))
	require.NoError(t, topUp.Insert())

	completed, credited, err := completeTopUp(tradeNo, PaymentProviderWaffo, topUpCompletionMetadata{})
	require.NoError(t, err)
	assert.True(t, credited)
	assert.Equal(t, common.TopUpStatusSuccess, completed.Status)
	requireWallet(t, DB, user.Id, 500, 75, 0)

	completed, credited, err = completeTopUp(tradeNo, PaymentProviderWaffo, topUpCompletionMetadata{})
	require.NoError(t, err)
	assert.False(t, credited)
	assert.Equal(t, common.TopUpStatusSuccess, completed.Status)
	requireWallet(t, DB, user.Id, 500, 75, 0)

	var ledgerCount int64
	require.NoError(t, DB.Model(&QuotaLedger{}).Where("event_key LIKE ?", "topup:"+tradeNo+":%").Count(&ledgerCount).Error)
	assert.Equal(t, int64(2), ledgerCount)
}

func TestCompleteTopUpLegacyPendingOrderFallsBackToPaidWithoutBonus(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&QuotaLedger{}))
	tradeNo := fmt.Sprintf("legacy-topup-%d", time.Now().UnixNano())
	user := &User{Username: tradeNo, Password: "password", Status: common.UserStatusEnabled, WalletVersion: CurrentWalletVersion}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, (&TopUp{
		UserId:          user.Id,
		Amount:          2,
		Money:           9.99,
		TradeNo:         tradeNo,
		PaymentMethod:   PaymentMethodWaffoPancake,
		PaymentProvider: PaymentProviderWaffoPancake,
		CreateTime:      common.GetTimestamp(),
		Status:          common.TopUpStatusPending,
	}).Insert())

	completed, credited, err := completeTopUp(tradeNo, PaymentProviderWaffoPancake, topUpCompletionMetadata{})
	require.NoError(t, err)
	assert.True(t, credited)
	assert.Equal(t, 2*int(common.QuotaPerUnit), completed.PrincipalQuota)
	assert.Zero(t, completed.PromoQuota)
	assert.Equal(t, CurrentTopUpSnapshotVersion, completed.SnapshotVersion)
	requireWallet(t, DB, user.Id, 2*int(common.QuotaPerUnit), 0, 0)
}
