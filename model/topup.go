package model

import (
	"errors"
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TopUp struct {
	Id              int     `json:"id"`
	UserId          int     `json:"user_id" gorm:"index"`
	Amount          int64   `json:"amount"`
	Money           float64 `json:"money"`
	TradeNo         string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string  `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	CreateTime      int64   `json:"create_time"`
	CompleteTime    int64   `json:"complete_time"`
	Status          string  `json:"status"`
	PrincipalQuota  int     `json:"principal_quota" gorm:"type:int;default:0"`
	PromoQuota      int     `json:"promo_quota" gorm:"type:int;default:0"`
	SnapshotVersion int     `json:"snapshot_version" gorm:"type:int;default:0"`
}

const CurrentTopUpSnapshotVersion = 1

const (
	PaymentMethodStripe       = "stripe"
	PaymentMethodCreem        = "creem"
	PaymentMethodWaffo        = "waffo"
	PaymentMethodWaffoPancake = "waffo_pancake"
	PaymentMethodBalance      = "balance"
)

const (
	PaymentProviderEpay         = "epay"
	PaymentProviderStripe       = "stripe"
	PaymentProviderCreem        = "creem"
	PaymentProviderWaffo        = "waffo"
	PaymentProviderWaffoPancake = "waffo_pancake"
	PaymentProviderBalance      = "balance"
)

var (
	ErrPaymentMethodMismatch = errors.New("payment method mismatch")
	ErrTopUpNotFound         = errors.New("topup not found")
	ErrTopUpStatusInvalid    = errors.New("topup status invalid")
)

type topUpCompletionMetadata struct {
	StripeCustomer string
	CustomerEmail  string
	PaymentMethod  string
}

func (topUp *TopUp) SetQuotaSnapshot(principalQuota int, promoQuota int) error {
	if principalQuota <= 0 || principalQuota > common.MaxQuota {
		return errors.New("无效的充值本金")
	}
	if promoQuota < 0 || promoQuota > common.MaxQuota-principalQuota {
		return errors.New("无效的充值赠送额度")
	}
	topUp.PrincipalQuota = principalQuota
	topUp.PromoQuota = promoQuota
	topUp.SnapshotVersion = CurrentTopUpSnapshotVersion
	return nil
}

func (topUp *TopUp) ensureQuotaSnapshot() error {
	if topUp.SnapshotVersion > 0 {
		return topUp.SetQuotaSnapshot(topUp.PrincipalQuota, topUp.PromoQuota)
	}

	var quotaDecimal decimal.Decimal
	if topUp.PaymentProvider == PaymentProviderCreem {
		quotaDecimal = decimal.NewFromInt(topUp.Amount)
	} else if topUp.PaymentProvider == PaymentProviderStripe {
		if math.IsNaN(topUp.Money) || math.IsInf(topUp.Money, 0) {
			return errors.New("无效的充值额度")
		}
		quotaDecimal = decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	} else {
		quotaDecimal = decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	}
	principalQuota, clamp := common.QuotaFromDecimalChecked(quotaDecimal)
	if clamp != nil {
		return errors.New("充值额度超出上限")
	}
	return topUp.SetQuotaSnapshot(principalQuota, 0)
}

func topUpWalletEventKey(tradeNo string, bucket WalletBucket) string {
	eventKey := "topup:" + tradeNo + ":" + string(bucket)
	if len(eventKey) <= 191 {
		return eventKey
	}
	return "topup:" + common.Sha1([]byte(tradeNo)) + ":" + string(bucket)
}

func completeTopUp(tradeNo string, expectedPaymentProvider string, metadata topUpCompletionMetadata) (*TopUp, bool, error) {
	if tradeNo == "" {
		return nil, false, errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	topUp := &TopUp{}
	credited := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}
		if metadata.PaymentMethod != "" {
			topUp.PaymentMethod = metadata.PaymentMethod
		}
		if err := topUp.ensureQuotaSnapshot(); err != nil {
			return err
		}

		if _, err := CreditWalletTx(tx, topUp.UserId, topUp.PrincipalQuota, WalletBucketPaid, topUpWalletEventKey(topUp.TradeNo, WalletBucketPaid)); err != nil {
			return err
		}
		if topUp.PromoQuota > 0 {
			if _, err := CreditWalletTx(tx, topUp.UserId, topUp.PromoQuota, WalletBucketPromo, topUpWalletEventKey(topUp.TradeNo, WalletBucketPromo)); err != nil {
				return err
			}
		}

		if metadata.StripeCustomer != "" {
			if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("stripe_customer", metadata.StripeCustomer).Error; err != nil {
				return err
			}
		}
		if metadata.CustomerEmail != "" {
			if err := tx.Model(&User{}).Where("id = ? AND (email = '' OR email IS NULL)", topUp.UserId).Update("email", metadata.CustomerEmail).Error; err != nil {
				return err
			}
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		credited = true
		return nil
	})
	if err != nil {
		return topUp, false, err
	}
	if credited {
		if err := InvalidateUserCache(topUp.UserId); err != nil {
			common.SysLog(fmt.Sprintf("failed to invalidate user cache after topup %s: %s", tradeNo, err.Error()))
		}
	}
	return topUp, credited, nil
}

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func UpdatePendingTopUpStatus(tradeNo string, expectedPaymentProvider string, targetStatus string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		topUp.Status = targetStatus
		return tx.Save(topUp).Error
	})
}

func Recharge(referenceId string, customerId string, callerIp string) (err error) {
	topUp, credited, err := completeTopUp(referenceId, PaymentProviderStripe, topUpCompletionMetadata{StripeCustomer: customerId})
	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if credited {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，赠送额度: %v，支付金额：%d", logger.FormatQuota(topUp.PrincipalQuota), logger.FormatQuota(topUp.PromoQuota), topUp.Amount), callerIp, topUp.PaymentMethod, PaymentMethodStripe)
	}
	return nil
}

// topUpQueryWindowSeconds 限制充值记录查询的时间窗口（秒）。
const topUpQueryWindowSeconds int64 = 30 * 24 * 60 * 60

// topUpQueryCutoff 返回允许查询的最早 create_time（秒级 Unix 时间戳）。
func topUpQueryCutoff() int64 {
	return common.GetTimestamp() - topUpQueryWindowSeconds
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cutoff := topUpQueryCutoff()

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, cutoff).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Where("user_id = ? AND create_time >= ?", userId, cutoff).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用，不限制时间窗口）
func GetAllTopUps(pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err = tx.Model(&TopUp{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// searchTopUpCountHardLimit 搜索充值记录时 COUNT 的安全上限，
// 防止对超大表执行无界 COUNT 触发 DoS。
const searchTopUpCountHardLimit = 10000

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// SearchAllTopUps 按订单号搜索全平台充值记录（管理员使用，不限制时间窗口）
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{})
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值
func ManualCompleteTopUp(tradeNo string, callerIp string) error {
	topUp, credited, err := completeTopUp(tradeNo, "", topUpCompletionMetadata{})
	if err != nil {
		return err
	}
	if credited {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("管理员补单成功，充值金额: %v，赠送额度: %v，支付金额：%f", logger.FormatQuota(topUp.PrincipalQuota), logger.FormatQuota(topUp.PromoQuota), topUp.Money), callerIp, topUp.PaymentMethod, "admin")
	}
	return nil
}

func RechargeEpay(tradeNo string, paymentMethod string, callerIp string) error {
	topUp, credited, err := completeTopUp(tradeNo, PaymentProviderEpay, topUpCompletionMetadata{PaymentMethod: paymentMethod})
	if err != nil {
		return err
	}
	if credited {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，赠送额度: %v，支付金额：%f", logger.LogQuota(topUp.PrincipalQuota), logger.LogQuota(topUp.PromoQuota), topUp.Money), callerIp, topUp.PaymentMethod, PaymentProviderEpay)
	}
	return nil
}

func RechargeCreem(referenceId string, customerEmail string, customerName string, callerIp string) (err error) {
	topUp, credited, err := completeTopUp(referenceId, PaymentProviderCreem, topUpCompletionMetadata{CustomerEmail: customerEmail})
	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if credited {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("使用Creem充值成功，充值额度: %v，赠送额度: %v，支付金额：%.2f", logger.FormatQuota(topUp.PrincipalQuota), logger.FormatQuota(topUp.PromoQuota), topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodCreem)
	}
	return nil
}

func RechargeWaffo(tradeNo string, callerIp string) (err error) {
	topUp, credited, err := completeTopUp(tradeNo, PaymentProviderWaffo, topUpCompletionMetadata{})
	if err != nil {
		common.SysError("waffo topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if credited {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("Waffo充值成功，充值额度: %v，赠送额度: %v，支付金额: %.2f", logger.FormatQuota(topUp.PrincipalQuota), logger.FormatQuota(topUp.PromoQuota), topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodWaffo)
	}
	return nil
}

func RechargeWaffoPancake(tradeNo string) (err error) {
	topUp, credited, err := completeTopUp(tradeNo, PaymentProviderWaffoPancake, topUpCompletionMetadata{})
	if err != nil {
		common.SysError("waffo pancake topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if credited {
		RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("Waffo Pancake充值成功，充值额度: %v，赠送额度: %v，支付金额: %.2f", logger.FormatQuota(topUp.PrincipalQuota), logger.FormatQuota(topUp.PromoQuota), topUp.Money))
	}
	return nil
}
