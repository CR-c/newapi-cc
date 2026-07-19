package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const CurrentWalletVersion = 2

type WalletBucket string

const (
	WalletBucketPaid          WalletBucket = "paid"
	WalletBucketPromo         WalletBucket = "promo"
	WalletBucketLegacyUnknown WalletBucket = "legacy_unknown"
)

const (
	walletOperationCredit     = "credit"
	walletOperationDebit      = "debit"
	walletOperationRefund     = "refund"
	walletOperationReclassify = "reclassify"
)

var (
	ErrInvalidWalletAmount           = errors.New("wallet amount must be positive")
	ErrInvalidWalletBucket           = errors.New("invalid wallet bucket")
	ErrInvalidWalletEventKey         = errors.New("wallet event key is required")
	ErrInsufficientWalletQuota       = errors.New("insufficient wallet quota")
	ErrWalletEventConflict           = errors.New("wallet event key conflicts with an existing mutation")
	ErrWalletBalanceCorrupt          = errors.New("wallet bucket total does not match user quota")
	ErrWalletQuotaOverflow           = errors.New("wallet quota exceeds supported range")
	ErrWalletSnapshotChanged         = errors.New("wallet balance snapshot changed")
	ErrInvalidWalletReclassification = errors.New("wallet reclassification must preserve total quota and clear legacy balance")
)

// WalletAllocation records the exact balance buckets used by a mutation.
// Values are always non-negative, including for debits.
type WalletAllocation struct {
	PaidQuota   int `json:"paid_quota"`
	PromoQuota  int `json:"promo_quota"`
	LegacyQuota int `json:"legacy_quota"`
}

func (a WalletAllocation) Total() int {
	return a.PaidQuota + a.PromoQuota + a.LegacyQuota
}

// QuotaLedger makes wallet mutations replayable. EventKey is globally unique:
// retrying the same mutation returns its original allocation without applying
// the balance change again.
type QuotaLedger struct {
	Id                       int64  `json:"id"`
	UserId                   int    `json:"user_id" gorm:"not null;index"`
	EventKey                 string `json:"event_key" gorm:"type:varchar(191);not null;uniqueIndex"`
	Operation                string `json:"operation" gorm:"type:varchar(16);not null;index"`
	Bucket                   string `json:"bucket" gorm:"type:varchar(32);not null;default:''"`
	Amount                   int    `json:"amount" gorm:"type:int;not null"`
	PaidQuota                int    `json:"paid_quota" gorm:"type:int;not null;default:0"`
	PromoQuota               int    `json:"promo_quota" gorm:"type:int;not null;default:0"`
	LegacyUnknownQuota       int    `json:"legacy_unknown_quota" gorm:"type:int;not null;default:0"`
	BeforePaidQuota          int    `json:"before_paid_quota" gorm:"type:int;not null;default:0"`
	BeforePromoQuota         int    `json:"before_promo_quota" gorm:"type:int;not null;default:0"`
	BeforeLegacyUnknownQuota int    `json:"before_legacy_unknown_quota" gorm:"type:int;not null;default:0"`
	Reason                   string `json:"reason" gorm:"type:varchar(255);not null;default:''"`
	OperatorId               int    `json:"operator_id" gorm:"type:int;not null;default:0;index"`
	CreatedAt                int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
}

func (ledger QuotaLedger) allocation() WalletAllocation {
	return WalletAllocation{
		PaidQuota:   ledger.PaidQuota,
		PromoQuota:  ledger.PromoQuota,
		LegacyQuota: ledger.LegacyUnknownQuota,
	}
}

func (ledger QuotaLedger) beforeAllocation() WalletAllocation {
	return WalletAllocation{
		PaidQuota:   ledger.BeforePaidQuota,
		PromoQuota:  ledger.BeforePromoQuota,
		LegacyQuota: ledger.BeforeLegacyUnknownQuota,
	}
}

// migrateUserWallets treats every pre-v2 balance as paid. Versioning keeps the
// migration idempotent and preserves later Root-managed paid/gift splits.
func migrateUserWallets() error {
	if err := DB.Unscoped().Model(&User{}).
		Where("wallet_version < ? AND quota < ?", CurrentWalletVersion, 0).
		Updates(map[string]interface{}{
			"quota": 0, "paid_quota": 0, "promo_quota": 0,
			"legacy_unknown_quota": 0, "wallet_version": CurrentWalletVersion,
		}).Error; err != nil {
		return err
	}
	return DB.Unscoped().Model(&User{}).
		Where("wallet_version < ? AND quota >= ?", CurrentWalletVersion, 0).
		Updates(map[string]interface{}{
			"paid_quota":           gorm.Expr("quota"),
			"promo_quota":          0,
			"legacy_unknown_quota": 0,
			"wallet_version":       CurrentWalletVersion,
		}).Error
}

func validateWalletMutation(amount int, eventKey string) error {
	if amount <= 0 {
		return ErrInvalidWalletAmount
	}
	if amount > math.MaxInt32 {
		return ErrWalletQuotaOverflow
	}
	if strings.TrimSpace(eventKey) == "" || len(eventKey) > 191 {
		return ErrInvalidWalletEventKey
	}
	return nil
}

func validWalletBucket(bucket WalletBucket) bool {
	return bucket == WalletBucketPaid || bucket == WalletBucketPromo || bucket == WalletBucketLegacyUnknown
}

func walletAllocationValid(allocation WalletAllocation) bool {
	if allocation.PaidQuota < 0 || allocation.PromoQuota < 0 || allocation.LegacyQuota < 0 {
		return false
	}
	total := int64(allocation.PaidQuota) + int64(allocation.PromoQuota) + int64(allocation.LegacyQuota)
	return total > 0 && total <= math.MaxInt32
}

func getWalletReplay(tx *gorm.DB, expected QuotaLedger) (WalletAllocation, bool, error) {
	var existing QuotaLedger
	err := tx.Where("event_key = ?", expected.EventKey).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return WalletAllocation{}, false, nil
	}
	if err != nil {
		return WalletAllocation{}, false, err
	}
	allocationSpecified := expected.PaidQuota != 0 || expected.PromoQuota != 0 || expected.LegacyUnknownQuota != 0
	if existing.UserId != expected.UserId || existing.Operation != expected.Operation ||
		existing.Bucket != expected.Bucket || existing.Amount != expected.Amount ||
		(allocationSpecified && (existing.PaidQuota != expected.PaidQuota || existing.PromoQuota != expected.PromoQuota ||
			existing.LegacyUnknownQuota != expected.LegacyUnknownQuota)) {
		return WalletAllocation{}, false, ErrWalletEventConflict
	}
	return existing.allocation(), true, nil
}

func loadWalletForUpdate(tx *gorm.DB, userID int) (*User, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id: %d", userID)
	}
	var user User
	if err := lockForUpdate(tx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	if user.WalletVersion < CurrentWalletVersion {
		if user.Quota < 0 {
			user.Quota = 0
		}
		user.PaidQuota = user.Quota
		user.PromoQuota = 0
		user.LegacyUnknownQuota = 0
		user.WalletVersion = CurrentWalletVersion
	}
	total := int64(user.PaidQuota) + int64(user.PromoQuota) + int64(user.LegacyUnknownQuota)
	if total != int64(user.Quota) || total < 0 || total > math.MaxInt32 {
		return nil, ErrWalletBalanceCorrupt
	}
	return &user, nil
}

func saveWallet(tx *gorm.DB, user *User) error {
	return tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"quota":                user.Quota,
		"paid_quota":           user.PaidQuota,
		"promo_quota":          user.PromoQuota,
		"legacy_unknown_quota": user.LegacyUnknownQuota,
		"wallet_version":       CurrentWalletVersion,
	}).Error
}

func runWalletTransaction(fn func(tx *gorm.DB) error) error {
	const maxSQLiteAttempts = 8
	for attempt := 0; ; attempt++ {
		err := DB.Transaction(fn)
		if err == nil || !common.UsingMainDatabase(common.DatabaseTypeSQLite) {
			return err
		}
		message := strings.ToLower(err.Error())
		locked := strings.Contains(message, "database is locked") ||
			strings.Contains(message, "database table is locked") ||
			strings.Contains(message, "database is deadlocked")
		if !locked || attempt+1 >= maxSQLiteAttempts {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * time.Millisecond)
	}
}

func CreditWalletTx(tx *gorm.DB, userID, amount int, bucket WalletBucket, eventKey string) (WalletAllocation, error) {
	if tx == nil {
		return WalletAllocation{}, errors.New("wallet transaction is nil")
	}
	if err := validateWalletMutation(amount, eventKey); err != nil {
		return WalletAllocation{}, err
	}
	if !validWalletBucket(bucket) {
		return WalletAllocation{}, ErrInvalidWalletBucket
	}
	allocation := WalletAllocation{}
	switch bucket {
	case WalletBucketPaid:
		allocation.PaidQuota = amount
	case WalletBucketPromo:
		allocation.PromoQuota = amount
	case WalletBucketLegacyUnknown:
		allocation.LegacyQuota = amount
	}
	ledger := QuotaLedger{
		UserId: userID, EventKey: eventKey, Operation: walletOperationCredit,
		Bucket: string(bucket), Amount: amount, PaidQuota: allocation.PaidQuota,
		PromoQuota: allocation.PromoQuota, LegacyUnknownQuota: allocation.LegacyQuota,
	}
	if replay, found, err := getWalletReplay(tx, ledger); found || err != nil {
		return replay, err
	}
	user, err := loadWalletForUpdate(tx, userID)
	if err != nil {
		return WalletAllocation{}, err
	}
	if replay, found, err := getWalletReplay(tx, ledger); found || err != nil {
		return replay, err
	}
	if int64(user.Quota)+int64(amount) > math.MaxInt32 {
		return WalletAllocation{}, ErrWalletQuotaOverflow
	}
	switch bucket {
	case WalletBucketPaid:
		user.PaidQuota += amount
	case WalletBucketPromo:
		user.PromoQuota += amount
	case WalletBucketLegacyUnknown:
		user.LegacyUnknownQuota += amount
	}
	user.Quota += amount
	if err := tx.Create(&ledger).Error; err != nil {
		return WalletAllocation{}, err
	}
	if err := saveWallet(tx, user); err != nil {
		return WalletAllocation{}, err
	}
	return allocation, nil
}

func CreditWallet(userID, amount int, bucket WalletBucket, eventKey string) (WalletAllocation, error) {
	var allocation WalletAllocation
	err := runWalletTransaction(func(tx *gorm.DB) error {
		var err error
		allocation, err = CreditWalletTx(tx, userID, amount, bucket, eventKey)
		return err
	})
	if err != nil {
		ledger := QuotaLedger{
			UserId: userID, EventKey: eventKey, Operation: walletOperationCredit,
			Bucket: string(bucket), Amount: amount, PaidQuota: allocation.PaidQuota,
			PromoQuota: allocation.PromoQuota, LegacyUnknownQuota: allocation.LegacyQuota,
		}
		if replay, found, replayErr := getWalletReplay(DB, ledger); found || replayErr != nil {
			allocation, err = replay, replayErr
		}
	}
	if err == nil {
		if cacheErr := invalidateUserCache(userID); cacheErr != nil {
			common.SysLog("failed to invalidate user cache after wallet credit: " + cacheErr.Error())
		}
	}
	return allocation, err
}

func DebitWalletTx(tx *gorm.DB, userID, amount int, eventKey string) (WalletAllocation, error) {
	if tx == nil {
		return WalletAllocation{}, errors.New("wallet transaction is nil")
	}
	if err := validateWalletMutation(amount, eventKey); err != nil {
		return WalletAllocation{}, err
	}
	baseLedger := QuotaLedger{UserId: userID, EventKey: eventKey, Operation: walletOperationDebit, Amount: amount}
	var existing QuotaLedger
	err := tx.Where("event_key = ?", eventKey).First(&existing).Error
	if err == nil {
		baseLedger.PaidQuota = existing.PaidQuota
		baseLedger.PromoQuota = existing.PromoQuota
		baseLedger.LegacyUnknownQuota = existing.LegacyUnknownQuota
		replay, _, replayErr := getWalletReplay(tx, baseLedger)
		return replay, replayErr
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return WalletAllocation{}, err
	}
	user, err := loadWalletForUpdate(tx, userID)
	if err != nil {
		return WalletAllocation{}, err
	}
	var replayLedger QuotaLedger
	if err := tx.Where("event_key = ?", eventKey).First(&replayLedger).Error; err == nil {
		baseLedger.PaidQuota = replayLedger.PaidQuota
		baseLedger.PromoQuota = replayLedger.PromoQuota
		baseLedger.LegacyUnknownQuota = replayLedger.LegacyUnknownQuota
		replay, _, replayErr := getWalletReplay(tx, baseLedger)
		return replay, replayErr
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return WalletAllocation{}, err
	}
	if user.Quota < amount {
		return WalletAllocation{}, ErrInsufficientWalletQuota
	}
	remaining := amount
	allocation := WalletAllocation{}
	allocation.PromoQuota = min(remaining, user.PromoQuota)
	remaining -= allocation.PromoQuota
	allocation.LegacyQuota = min(remaining, user.LegacyUnknownQuota)
	remaining -= allocation.LegacyQuota
	allocation.PaidQuota = remaining

	user.PromoQuota -= allocation.PromoQuota
	user.LegacyUnknownQuota -= allocation.LegacyQuota
	user.PaidQuota -= allocation.PaidQuota
	user.Quota -= amount
	ledger := QuotaLedger{
		UserId: userID, EventKey: eventKey, Operation: walletOperationDebit, Amount: amount,
		PaidQuota: allocation.PaidQuota, PromoQuota: allocation.PromoQuota,
		LegacyUnknownQuota: allocation.LegacyQuota,
	}
	if err := tx.Create(&ledger).Error; err != nil {
		return WalletAllocation{}, err
	}
	if err := saveWallet(tx, user); err != nil {
		return WalletAllocation{}, err
	}
	return allocation, nil
}

func DebitWallet(userID, amount int, eventKey string) (WalletAllocation, error) {
	var allocation WalletAllocation
	err := runWalletTransaction(func(tx *gorm.DB) error {
		var err error
		allocation, err = DebitWalletTx(tx, userID, amount, eventKey)
		return err
	})
	if err != nil {
		baseLedger := QuotaLedger{UserId: userID, EventKey: eventKey, Operation: walletOperationDebit, Amount: amount}
		if replay, found, replayErr := getWalletReplay(DB, baseLedger); found || replayErr != nil {
			allocation, err = replay, replayErr
		}
	}
	if err == nil {
		if cacheErr := invalidateUserCache(userID); cacheErr != nil {
			common.SysLog("failed to invalidate user cache after wallet debit: " + cacheErr.Error())
		}
	}
	return allocation, err
}

func RefundWalletTx(tx *gorm.DB, userID int, allocation WalletAllocation, eventKey string) (WalletAllocation, error) {
	if tx == nil {
		return WalletAllocation{}, errors.New("wallet transaction is nil")
	}
	if !walletAllocationValid(allocation) {
		return WalletAllocation{}, ErrInvalidWalletAmount
	}
	amount := allocation.Total()
	if err := validateWalletMutation(amount, eventKey); err != nil {
		return WalletAllocation{}, err
	}
	ledger := QuotaLedger{
		UserId: userID, EventKey: eventKey, Operation: walletOperationRefund, Amount: amount,
		PaidQuota: allocation.PaidQuota, PromoQuota: allocation.PromoQuota,
		LegacyUnknownQuota: allocation.LegacyQuota,
	}
	if replay, found, err := getWalletReplay(tx, ledger); found || err != nil {
		return replay, err
	}
	user, err := loadWalletForUpdate(tx, userID)
	if err != nil {
		return WalletAllocation{}, err
	}
	if replay, found, err := getWalletReplay(tx, ledger); found || err != nil {
		return replay, err
	}
	if int64(user.Quota)+int64(amount) > math.MaxInt32 {
		return WalletAllocation{}, ErrWalletQuotaOverflow
	}
	user.PaidQuota += allocation.PaidQuota
	user.PromoQuota += allocation.PromoQuota
	user.LegacyUnknownQuota += allocation.LegacyQuota
	user.Quota += amount
	if err := tx.Create(&ledger).Error; err != nil {
		return WalletAllocation{}, err
	}
	if err := saveWallet(tx, user); err != nil {
		return WalletAllocation{}, err
	}
	return allocation, nil
}

func RefundWallet(userID int, allocation WalletAllocation, eventKey string) (WalletAllocation, error) {
	var refunded WalletAllocation
	err := runWalletTransaction(func(tx *gorm.DB) error {
		var err error
		refunded, err = RefundWalletTx(tx, userID, allocation, eventKey)
		return err
	})
	if err != nil {
		ledger := QuotaLedger{
			UserId: userID, EventKey: eventKey, Operation: walletOperationRefund, Amount: allocation.Total(),
			PaidQuota: allocation.PaidQuota, PromoQuota: allocation.PromoQuota,
			LegacyUnknownQuota: allocation.LegacyQuota,
		}
		if replay, found, replayErr := getWalletReplay(DB, ledger); found || replayErr != nil {
			refunded, err = replay, replayErr
		}
	}
	if err == nil {
		if cacheErr := invalidateUserCache(userID); cacheErr != nil {
			common.SysLog("failed to invalidate user cache after wallet refund: " + cacheErr.Error())
		}
	}
	return refunded, err
}

func getWalletReclassificationReplay(tx *gorm.DB, expected QuotaLedger) (WalletAllocation, WalletAllocation, bool, error) {
	var existing QuotaLedger
	err := tx.Where("event_key = ?", expected.EventKey).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return WalletAllocation{}, WalletAllocation{}, false, nil
	}
	if err != nil {
		return WalletAllocation{}, WalletAllocation{}, false, err
	}
	if existing.UserId != expected.UserId || existing.Operation != walletOperationReclassify ||
		existing.Bucket != expected.Bucket || existing.Amount != expected.Amount ||
		existing.PaidQuota != expected.PaidQuota || existing.PromoQuota != expected.PromoQuota ||
		existing.LegacyUnknownQuota != expected.LegacyUnknownQuota ||
		existing.BeforePaidQuota != expected.BeforePaidQuota ||
		existing.BeforePromoQuota != expected.BeforePromoQuota ||
		existing.BeforeLegacyUnknownQuota != expected.BeforeLegacyUnknownQuota ||
		existing.Reason != expected.Reason || existing.OperatorId != expected.OperatorId {
		return WalletAllocation{}, WalletAllocation{}, false, ErrWalletEventConflict
	}
	return existing.beforeAllocation(), existing.allocation(), true, nil
}

// ReclassifyWallet lets Root redistribute an existing wallet between paid and
// gift balances. Total quota is preserved and legacy attribution is cleared.
func ReclassifyWallet(userID int, expected, target WalletAllocation, eventKey, reason string, operatorID int) (WalletAllocation, WalletAllocation, error) {
	if !walletAllocationValid(expected) || !walletAllocationValid(target) || expected.Total() != target.Total() {
		return WalletAllocation{}, WalletAllocation{}, ErrInvalidWalletAmount
	}
	if target.LegacyQuota != 0 {
		return WalletAllocation{}, WalletAllocation{}, ErrInvalidWalletReclassification
	}
	deltaPaid := int64(target.PaidQuota) - int64(expected.PaidQuota)
	deltaPromo := int64(target.PromoQuota) - int64(expected.PromoQuota)
	deltaLegacy := int64(target.LegacyQuota) - int64(expected.LegacyQuota)
	moved := int64(0)
	for _, delta := range []int64{deltaPaid, deltaPromo, deltaLegacy} {
		if delta < 0 {
			moved -= delta
		} else {
			moved += delta
		}
	}
	moved /= 2
	if moved <= 0 || moved > math.MaxInt32 {
		return WalletAllocation{}, WalletAllocation{}, ErrInvalidWalletReclassification
	}
	reason = strings.TrimSpace(reason)
	reasonLength := utf8.RuneCountInString(reason)
	if reasonLength < 3 || reasonLength > 200 || operatorID <= 0 {
		return WalletAllocation{}, WalletAllocation{}, ErrInvalidWalletReclassification
	}
	if err := validateWalletMutation(int(moved), eventKey); err != nil {
		return WalletAllocation{}, WalletAllocation{}, err
	}
	ledger := QuotaLedger{
		UserId: userID, EventKey: eventKey, Operation: walletOperationReclassify,
		Bucket: "manual", Amount: int(moved),
		PaidQuota: target.PaidQuota, PromoQuota: target.PromoQuota, LegacyUnknownQuota: target.LegacyQuota,
		BeforePaidQuota: expected.PaidQuota, BeforePromoQuota: expected.PromoQuota,
		BeforeLegacyUnknownQuota: expected.LegacyQuota, Reason: reason, OperatorId: operatorID,
	}
	if before, after, found, err := getWalletReclassificationReplay(DB, ledger); found || err != nil {
		return before, after, err
	}

	err := runWalletTransaction(func(tx *gorm.DB) error {
		if _, _, found, err := getWalletReclassificationReplay(tx, ledger); found || err != nil {
			return err
		}
		user, err := loadWalletForUpdate(tx, userID)
		if err != nil {
			return err
		}
		current := WalletAllocation{
			PaidQuota: user.PaidQuota, PromoQuota: user.PromoQuota,
			LegacyQuota: user.LegacyUnknownQuota,
		}
		if current != expected || user.Quota != expected.Total() {
			return ErrWalletSnapshotChanged
		}
		if _, _, found, err := getWalletReclassificationReplay(tx, ledger); found || err != nil {
			return err
		}
		if err := tx.Create(&ledger).Error; err != nil {
			return err
		}
		user.PaidQuota = target.PaidQuota
		user.PromoQuota = target.PromoQuota
		user.LegacyUnknownQuota = target.LegacyQuota
		return saveWallet(tx, user)
	})
	if err != nil {
		if before, after, found, replayErr := getWalletReclassificationReplay(DB, ledger); found || replayErr != nil {
			return before, after, replayErr
		}
	}
	if err != nil {
		return WalletAllocation{}, WalletAllocation{}, err
	}
	if cacheErr := invalidateUserCache(userID); cacheErr != nil {
		common.SysLog("failed to invalidate user cache after wallet reclassification: " + cacheErr.Error())
	}
	return expected, target, nil
}

// OverrideWalletQuota atomically moves a wallet to targetQuota. Increases are
// attributed to creditBucket; decreases retain the normal promo -> legacy -> paid order.
func OverrideWalletQuota(userID, targetQuota int, creditBucket WalletBucket, eventKey string) (int, WalletAllocation, error) {
	if targetQuota < 0 || targetQuota > common.MaxQuota {
		return 0, WalletAllocation{}, ErrWalletQuotaOverflow
	}
	if !validWalletBucket(creditBucket) {
		return 0, WalletAllocation{}, ErrInvalidWalletBucket
	}
	oldQuota := 0
	allocation := WalletAllocation{}
	err := runWalletTransaction(func(tx *gorm.DB) error {
		user, err := loadWalletForUpdate(tx, userID)
		if err != nil {
			return err
		}
		oldQuota = user.Quota
		delta := targetQuota - oldQuota
		switch {
		case delta > 0:
			allocation, err = CreditWalletTx(tx, userID, delta, creditBucket, eventKey)
		case delta < 0:
			allocation, err = DebitWalletTx(tx, userID, -delta, eventKey)
		}
		return err
	})
	if err == nil {
		if cacheErr := invalidateUserCache(userID); cacheErr != nil {
			common.SysLog("failed to invalidate user cache after wallet override: " + cacheErr.Error())
		}
	}
	return oldQuota, allocation, err
}
