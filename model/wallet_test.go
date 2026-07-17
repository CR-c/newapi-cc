package model

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupWalletTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=30000", dbName)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(8)
	require.NoError(t, db.AutoMigrate(&User{}, &QuotaLedger{}, &Log{}))

	originalDB := DB
	originalLogDB := LOG_DB
	DB, LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		require.NoError(t, sqlDB.Close())
	})
	return db
}

func requireWallet(t *testing.T, db *gorm.DB, userID int, paid, promo, legacy int) User {
	t.Helper()
	var user User
	require.NoError(t, db.First(&user, userID).Error)
	assert.Equal(t, paid, user.PaidQuota)
	assert.Equal(t, promo, user.PromoQuota)
	assert.Equal(t, legacy, user.LegacyUnknownQuota)
	assert.Equal(t, paid+promo+legacy, user.Quota)
	return user
}

func TestMigrateUserWalletsMovesLegacyQuotaExactlyOnce(t *testing.T) {
	db := setupWalletTestDB(t)
	legacy := User{Username: "legacy", Password: "password", Quota: 900, AffCode: "legacy"}
	initialized := User{
		Username: "initialized", Password: "password", Quota: 600,
		PaidQuota: 500, PromoQuota: 100, WalletVersion: CurrentWalletVersion, AffCode: "initialized",
	}
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Create(&initialized).Error)

	require.NoError(t, migrateUserWallets())
	require.NoError(t, migrateUserWallets())

	migrated := requireWallet(t, db, legacy.Id, 0, 0, 900)
	assert.Equal(t, CurrentWalletVersion, migrated.WalletVersion)
	unchanged := requireWallet(t, db, initialized.Id, 500, 100, 0)
	assert.Equal(t, CurrentWalletVersion, unchanged.WalletVersion)
}

func TestMigrateUserWalletsNormalizesNegativeLegacyBalance(t *testing.T) {
	db := setupWalletTestDB(t)
	user := User{Username: "negative-legacy", Password: "password", Quota: -25}
	require.NoError(t, db.Create(&user).Error)

	require.NoError(t, migrateUserWallets())
	migrated := requireWallet(t, db, user.Id, 0, 0, 0)
	assert.Equal(t, CurrentWalletVersion, migrated.WalletVersion)
}

func TestWalletDebitUsesPromoThenLegacyThenPaidAndRefundsExactly(t *testing.T) {
	db := setupWalletTestDB(t)
	user := User{
		Username: "wallet-user", Password: "password", Quota: 180,
		PaidQuota: 100, PromoQuota: 30, LegacyUnknownQuota: 50,
		WalletVersion: CurrentWalletVersion,
	}
	require.NoError(t, db.Create(&user).Error)

	allocation, err := DebitWallet(user.Id, 120, "consume:req-1")
	require.NoError(t, err)
	assert.Equal(t, WalletAllocation{PaidQuota: 40, PromoQuota: 30, LegacyQuota: 50}, allocation)
	requireWallet(t, db, user.Id, 60, 0, 0)

	refunded, err := RefundWallet(user.Id, allocation, "refund:req-1")
	require.NoError(t, err)
	assert.Equal(t, allocation, refunded)
	requireWallet(t, db, user.Id, 100, 30, 50)
}

func TestWalletCreditAndDebitAreIdempotentByEventKey(t *testing.T) {
	db := setupWalletTestDB(t)
	user := User{Username: "idempotent", Password: "password", WalletVersion: CurrentWalletVersion}
	require.NoError(t, db.Create(&user).Error)

	firstCredit, err := CreditWallet(user.Id, 80, WalletBucketPromo, "credit:gift-1")
	require.NoError(t, err)
	secondCredit, err := CreditWallet(user.Id, 80, WalletBucketPromo, "credit:gift-1")
	require.NoError(t, err)
	assert.Equal(t, firstCredit, secondCredit)
	requireWallet(t, db, user.Id, 0, 80, 0)

	firstDebit, err := DebitWallet(user.Id, 50, "consume:req-2")
	require.NoError(t, err)
	secondDebit, err := DebitWallet(user.Id, 50, "consume:req-2")
	require.NoError(t, err)
	assert.Equal(t, firstDebit, secondDebit)
	requireWallet(t, db, user.Id, 0, 30, 0)

	var ledgerCount int64
	require.NoError(t, db.Model(&QuotaLedger{}).Count(&ledgerCount).Error)
	assert.Equal(t, int64(2), ledgerCount)
}

func TestWalletConcurrentReplayAppliesEventOnce(t *testing.T) {
	db := setupWalletTestDB(t)
	user := User{Username: "concurrent-replay", Password: "password", WalletVersion: CurrentWalletVersion}
	require.NoError(t, db.Create(&user).Error)

	const callers = 16
	start := make(chan struct{})
	results := make(chan WalletAllocation, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for range callers {
		go func() {
			defer wg.Done()
			<-start
			allocation, err := CreditWallet(user.Id, 80, WalletBucketPromo, "credit:concurrent-gift")
			results <- allocation
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	for allocation := range results {
		assert.Equal(t, WalletAllocation{PromoQuota: 80}, allocation)
	}
	requireWallet(t, db, user.Id, 0, 80, 0)

	var ledgerCount int64
	require.NoError(t, db.Model(&QuotaLedger{}).Where("event_key = ?", "credit:concurrent-gift").Count(&ledgerCount).Error)
	assert.Equal(t, int64(1), ledgerCount)
}

func TestWalletRejectsInsufficientQuotaWithoutMutation(t *testing.T) {
	db := setupWalletTestDB(t)
	user := User{
		Username: "insufficient", Password: "password", Quota: 10,
		PaidQuota: 10, WalletVersion: CurrentWalletVersion,
	}
	require.NoError(t, db.Create(&user).Error)

	_, err := DebitWallet(user.Id, 11, "consume:too-much")
	require.ErrorIs(t, err, ErrInsufficientWalletQuota)
	requireWallet(t, db, user.Id, 10, 0, 0)

	var ledgerCount int64
	require.NoError(t, db.Model(&QuotaLedger{}).Count(&ledgerCount).Error)
	assert.Zero(t, ledgerCount)
}

func TestWalletMutationRollsBackWithCallerTransaction(t *testing.T) {
	db := setupWalletTestDB(t)
	user := User{Username: "rollback", Password: "password", WalletVersion: CurrentWalletVersion}
	require.NoError(t, db.Create(&user).Error)

	errRollback := errors.New("force rollback")
	err := db.Transaction(func(tx *gorm.DB) error {
		allocation, creditErr := CreditWalletTx(tx, user.Id, 70, WalletBucketPaid, "credit:rollback")
		require.NoError(t, creditErr)
		assert.Equal(t, WalletAllocation{PaidQuota: 70}, allocation)
		return errRollback
	})
	require.ErrorIs(t, err, errRollback)
	requireWallet(t, db, user.Id, 0, 0, 0)

	var ledgerCount int64
	require.NoError(t, db.Model(&QuotaLedger{}).Count(&ledgerCount).Error)
	assert.Zero(t, ledgerCount)
}

func TestWalletEventKeyCannotBeReusedForDifferentMutation(t *testing.T) {
	db := setupWalletTestDB(t)
	user := User{Username: "conflict", Password: "password", WalletVersion: CurrentWalletVersion}
	require.NoError(t, db.Create(&user).Error)

	_, err := CreditWallet(user.Id, 20, WalletBucketPromo, "event:same")
	require.NoError(t, err)
	_, err = CreditWallet(user.Id, 30, WalletBucketPromo, "event:same")
	require.ErrorIs(t, err, ErrWalletEventConflict)
	requireWallet(t, db, user.Id, 0, 20, 0)
}

func TestInsertedUserReceivesRegistrationQuotaAsPromo(t *testing.T) {
	db := setupWalletTestDB(t)
	originalQuota := common.QuotaForNewUser
	common.QuotaForNewUser = 123
	t.Cleanup(func() { common.QuotaForNewUser = originalQuota })

	user := User{Username: "new-wallet", Password: "password123", Status: common.UserStatusEnabled}
	require.NoError(t, user.Insert(0))
	inserted := requireWallet(t, db, user.Id, 0, 123, 0)
	assert.Equal(t, CurrentWalletVersion, inserted.WalletVersion)
}

func TestUserBaseWritesRoleToContext(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)
	user := User{Id: 7, Username: "admin", Role: common.RoleAdminUser}
	base := user.ToBaseUser()
	base.WriteContext(ctx)

	assert.Equal(t, common.RoleAdminUser, common.GetContextKeyInt(ctx, constant.ContextKeyUserRole))
}

func TestWalletAllocationJSONContract(t *testing.T) {
	encoded, err := common.Marshal(WalletAllocation{PaidQuota: 1, PromoQuota: 2, LegacyQuota: 3})
	require.NoError(t, err)
	assert.JSONEq(t, `{"paid_quota":1,"promo_quota":2,"legacy_quota":3}`, string(encoded))
}

func TestTransferAffQuotaCreditsPromoWallet(t *testing.T) {
	db := setupWalletTestDB(t)
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 100
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })
	user := User{
		Username: "affiliate", Password: "password", AffQuota: 200,
		WalletVersion: CurrentWalletVersion,
	}
	require.NoError(t, db.Create(&user).Error)

	require.NoError(t, user.TransferAffQuotaToQuota(100))
	stored := requireWallet(t, db, user.Id, 0, 100, 0)
	assert.Equal(t, 100, stored.AffQuota)

	var ledger QuotaLedger
	require.NoError(t, db.Where("user_id = ? AND operation = ?", user.Id, walletOperationCredit).First(&ledger).Error)
	assert.Equal(t, string(WalletBucketPromo), ledger.Bucket)
}

func TestOverrideWalletQuotaComputesDeltaUnderLock(t *testing.T) {
	db := setupWalletTestDB(t)
	user := User{
		Username: "override", Password: "password", Quota: 80,
		PaidQuota: 50, PromoQuota: 30, WalletVersion: CurrentWalletVersion,
	}
	require.NoError(t, db.Create(&user).Error)

	oldQuota, allocation, err := OverrideWalletQuota(user.Id, 120, WalletBucketPromo, "override:increase")
	require.NoError(t, err)
	assert.Equal(t, 80, oldQuota)
	assert.Equal(t, WalletAllocation{PromoQuota: 40}, allocation)
	requireWallet(t, db, user.Id, 50, 70, 0)

	oldQuota, allocation, err = OverrideWalletQuota(user.Id, 20, WalletBucketPromo, "override:decrease")
	require.NoError(t, err)
	assert.Equal(t, 120, oldQuota)
	assert.Equal(t, WalletAllocation{PaidQuota: 30, PromoQuota: 70}, allocation)
	requireWallet(t, db, user.Id, 20, 0, 0)
}
