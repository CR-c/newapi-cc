package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSearchRedemptionsFiltersAndPaginates(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})

	now := common.GetTimestamp()
	redemptions := []Redemption{
		{Id: 1, Name: "alpha-active", Key: "00000000000000000000000000000001", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: 0},
		{Id: 2, Name: "alpha-future", Key: "00000000000000000000000000000002", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now + 3600},
		{Id: 3, Name: "alpha-expired", Key: "00000000000000000000000000000003", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now - 10},
		{Id: 4, Name: "beta-disabled", Key: "00000000000000000000000000000004", Status: common.RedemptionCodeStatusDisabled, ExpiredTime: 0},
		{Id: 5, Name: "beta-used", Key: "00000000000000000000000000000005", Status: common.RedemptionCodeStatusUsed, ExpiredTime: 0},
	}
	require.NoError(t, DB.Create(&redemptions).Error)

	tests := []struct {
		name      string
		keyword   string
		status    string
		startIdx  int
		num       int
		wantTotal int64
		wantIds   []int
	}{
		{
			name:      "no filters returns all rows",
			num:       10,
			wantTotal: 5,
			wantIds:   []int{5, 4, 3, 2, 1},
		},
		{
			name:      "keyword filters by name prefix",
			keyword:   "alpha",
			num:       10,
			wantTotal: 3,
			wantIds:   []int{3, 2, 1},
		},
		{
			name:      "enabled status excludes expired rows",
			status:    "1",
			num:       10,
			wantTotal: 2,
			wantIds:   []int{2, 1},
		},
		{
			name:      "expired status returns enabled expired rows",
			status:    "expired",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{3},
		},
		{
			name:      "disabled status",
			status:    "2",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{4},
		},
		{
			name:      "used status",
			status:    "3",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{5},
		},
		{
			name:      "pagination keeps unpaged total",
			startIdx:  1,
			num:       2,
			wantTotal: 5,
			wantIds:   []int{4, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, total, err := SearchRedemptions(tt.keyword, tt.status, tt.startIdx, tt.num)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			gotIds := make([]int, 0, len(rows))
			for _, row := range rows {
				gotIds = append(gotIds, row.Id)
			}
			assert.Equal(t, tt.wantIds, gotIds)
		})
	}
}

func setupRedeemFixture(t *testing.T, quota int, fundingSource string) (userId int, key string) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&QuotaLedger{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&QuotaLedger{}).Error)
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
	})

	user := &User{Username: "redeem-user", Password: "password", Status: common.UserStatusEnabled, Quota: 0}
	require.NoError(t, DB.Create(user).Error)

	key = "10000000000000000000000000000001"
	redemption := &Redemption{
		Name:          "redeem-test",
		Key:           key,
		Status:        common.RedemptionCodeStatusEnabled,
		Quota:         quota,
		FundingSource: fundingSource,
		CreatedTime:   common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(redemption).Error)
	return user.Id, key
}

func TestRedeemCreditsQuotaExactlyOnce(t *testing.T) {
	userId, key := setupRedeemFixture(t, 500, RedemptionFundingSourcePaid)

	quota, err := Redeem(key, userId)
	require.NoError(t, err)
	assert.Equal(t, 500, quota)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)
	assert.Equal(t, 500, user.PaidQuota)
	assert.Zero(t, user.PromoQuota)
	assert.Zero(t, user.LegacyUnknownQuota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "name = ?", "redeem-test").Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
	assert.Equal(t, userId, redemption.UsedUserId)

	// Redeeming the same code again must fail and must not credit quota.
	_, err = Redeem(key, userId)
	require.Error(t, err)
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)
}

// Exactly one of several concurrent redeems of the same code may win, and
// quota must be credited exactly once.
func TestRedeemConcurrentSingleSuccess(t *testing.T) {
	userId, key := setupRedeemFixture(t, 300, RedemptionFundingSourcePaid)

	const goroutines = 5
	successes := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			if _, err := Redeem(key, userId); err == nil {
				successes[idx] = true
			}
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range successes {
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent redeem should succeed")

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 300, user.Quota, "quota must be credited exactly once")
	assert.Equal(t, 300, user.PaidQuota)
	assert.Zero(t, user.PromoQuota)
}

func TestRedeemCreditsConfiguredFundingBucket(t *testing.T) {
	tests := []struct {
		name          string
		fundingSource string
		want          WalletAllocation
	}{
		{name: "paid card", fundingSource: RedemptionFundingSourcePaid, want: WalletAllocation{PaidQuota: 400}},
		{name: "gift code", fundingSource: RedemptionFundingSourcePromo, want: WalletAllocation{PromoQuota: 400}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, key := setupRedeemFixture(t, 400, tt.fundingSource)

			quota, err := Redeem(key, userID)
			require.NoError(t, err)
			assert.Equal(t, 400, quota)
			requireWallet(t, DB, userID, tt.want.PaidQuota, tt.want.PromoQuota, 0)
		})
	}
}

func TestRedemptionFundingSourceValidation(t *testing.T) {
	assert.NoError(t, ValidateNewRedemptionFundingSource(RedemptionFundingSourcePaid))
	assert.NoError(t, ValidateNewRedemptionFundingSource(RedemptionFundingSourcePromo))
	for _, source := range []string{"", RedemptionFundingSourceLegacyUnknown, "cash", "PAID"} {
		assert.Error(t, ValidateNewRedemptionFundingSource(source))
	}
}

func TestInsertRedemptionsIsAtomic(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:redemption_batch_atomic?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Redemption{}))

	originalDB := DB
	DB = db
	t.Cleanup(func() { DB = originalDB })

	redemptions := []Redemption{
		{Name: "batch", Key: "duplicate-redemption-key", FundingSource: RedemptionFundingSourcePaid},
		{Name: "batch", Key: "duplicate-redemption-key", FundingSource: RedemptionFundingSourcePromo},
	}
	require.Error(t, InsertRedemptions(redemptions))

	var count int64
	require.NoError(t, db.Model(&Redemption{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestRedemptionMigrationMovesEveryExistingRowToPaidExactlyOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:redemption_source_migration?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Redemption{}))
	require.NoError(t, db.Create(&Redemption{Id: 1, Key: "paid-code", FundingSource: RedemptionFundingSourcePaid}).Error)
	require.NoError(t, db.Create(&Redemption{Id: 2, Key: "promo-code", FundingSource: RedemptionFundingSourcePromo}).Error)
	require.NoError(t, db.Create(&Redemption{Id: 3, Key: "legacy-code", FundingSource: RedemptionFundingSourceLegacyUnknown}).Error)
	require.NoError(t, db.Create(&Redemption{Id: 4, Key: "invalid-code", FundingSource: "cash"}).Error)
	require.NoError(t, db.Delete(&Redemption{}, 4).Error)

	originalDB := DB
	DB = db
	t.Cleanup(func() { DB = originalDB })
	require.NoError(t, migrateRedemptionFundingSources())
	require.NoError(t, db.Create(&Redemption{
		Id: 5, Key: "new-gift-code", FundingSource: RedemptionFundingSourcePromo,
		FundingVersion: CurrentRedemptionFundingVersion,
	}).Error)
	require.NoError(t, migrateRedemptionFundingSources())

	var redemptions []Redemption
	require.NoError(t, db.Unscoped().Order("id").Find(&redemptions).Error)
	require.Len(t, redemptions, 5)
	for _, redemption := range redemptions[:4] {
		assert.Equal(t, RedemptionFundingSourcePaid, redemption.FundingSource)
		assert.Equal(t, CurrentRedemptionFundingVersion, redemption.FundingVersion)
	}
	assert.Equal(t, RedemptionFundingSourcePromo, redemptions[4].FundingSource)
	assert.Equal(t, CurrentRedemptionFundingVersion, redemptions[4].FundingVersion)
}

func TestRedeemRejectsInvalidStoredFundingSourceWithoutConsumingCode(t *testing.T) {
	for _, source := range []string{RedemptionFundingSourceLegacyUnknown, "cash"} {
		t.Run(source, func(t *testing.T) {
			userID, key := setupRedeemFixture(t, 300, source)

			_, err := Redeem(key, userID)
			require.Error(t, err)
			requireWallet(t, DB, userID, 0, 0, 0)

			var redemption Redemption
			require.NoError(t, DB.Where("key = ?", key).First(&redemption).Error)
			assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
		})
	}
}

func TestNonRootMutationsRejectStaleLegacySourceAfterPaidResolution(t *testing.T) {
	_, key := setupRedeemFixture(t, 300, RedemptionFundingSourceLegacyUnknown)

	var stale Redemption
	require.NoError(t, DB.Where("key = ?", key).First(&stale).Error)
	require.NoError(t, DB.Model(&Redemption{}).
		Where("id = ? AND funding_source = ?", stale.Id, RedemptionFundingSourceLegacyUnknown).
		Update("funding_source", RedemptionFundingSourcePaid).Error)

	stale.Name = "stale update"
	require.ErrorIs(t,
		stale.UpdateDetails(common.RedemptionCodeStatusEnabled, RedemptionFundingSourceLegacyUnknown),
		ErrRedemptionStateChanged,
	)
	require.ErrorIs(t,
		UpdateRedemptionStatus(
			stale.Id,
			common.RedemptionCodeStatusEnabled,
			common.RedemptionCodeStatusDisabled,
			RedemptionFundingSourceLegacyUnknown,
			false,
		),
		ErrRedemptionStateChanged,
	)
	require.ErrorIs(t,
		DeleteRedemptionById(stale.Id, RedemptionFundingSourceLegacyUnknown, false),
		ErrRedemptionStateChanged,
	)

	var stored Redemption
	require.NoError(t, DB.First(&stored, stale.Id).Error)
	assert.Equal(t, RedemptionFundingSourcePaid, stored.FundingSource)
	assert.NotEqual(t, "stale update", stored.Name)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, stored.Status)
}

func TestUsedRedemptionStatusIsTerminal(t *testing.T) {
	_, key := setupRedeemFixture(t, 300, RedemptionFundingSourcePromo)
	var redemption Redemption
	require.NoError(t, DB.Where("key = ?", key).First(&redemption).Error)
	require.NoError(t, DB.Model(&Redemption{}).Where("id = ?", redemption.Id).
		Update("status", common.RedemptionCodeStatusUsed).Error)

	err := UpdateRedemptionStatus(
		redemption.Id,
		common.RedemptionCodeStatusUsed,
		common.RedemptionCodeStatusEnabled,
		RedemptionFundingSourcePromo,
		true,
	)
	require.ErrorIs(t, err, ErrRedemptionStateChanged)
}
