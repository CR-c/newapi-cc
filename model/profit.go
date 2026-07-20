package model

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	profitMicrosPerUnit             int64 = 1_000_000
	currentProfitAttributionVersion       = 1
)

var profitBackfillMutex sync.Mutex
var profitEpochMutex sync.RWMutex

var ErrProfitBackfillInProgress = errors.New("profit backfill is already in progress")

type ModelCostRule struct {
	Id               int64          `json:"id" gorm:"primaryKey"`
	ModelName        string         `json:"model_name" gorm:"type:varchar(191);index:idx_model_cost_rule,priority:1;uniqueIndex:idx_model_cost_rule_version,priority:1"`
	PurchasePriceCNY float64        `json:"purchase_price_cny"`
	CostTiers        ModelCostTiers `json:"cost_tiers" gorm:"type:text"`
	Version          int            `json:"version" gorm:"index:idx_model_cost_rule,priority:2;uniqueIndex:idx_model_cost_rule_version,priority:2"`
	Enabled          bool           `json:"enabled" gorm:"index"`
	EffectiveFrom    int64          `json:"effective_from" gorm:"index"`
	EffectiveTo      int64          `json:"effective_to" gorm:"index"`
	CreatedAt        int64          `json:"created_at"`
	UpdatedAt        int64          `json:"updated_at"`

	UpstreamPriceUSD float64 `json:"-" gorm:"column:upstream_price_usd;->;-:migration"`
	ExchangeRate     float64 `json:"-" gorm:"column:exchange_rate;->;-:migration"`
	Discount         float64 `json:"-" gorm:"column:discount;->;-:migration"`
}

type ModelCostTier struct {
	Key              string  `json:"key"`
	Label            string  `json:"label"`
	PurchasePriceCNY float64 `json:"purchase_price_cny"`
}

type ModelCostTiers []ModelCostTier

func (tiers ModelCostTiers) Value() (driver.Value, error) {
	if len(tiers) == 0 {
		return nil, nil
	}
	data, err := common.Marshal(tiers)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (tiers *ModelCostTiers) Scan(value interface{}) error {
	if value == nil {
		*tiers = nil
		return nil
	}
	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = append([]byte(nil), typed...)
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("unsupported cost tiers database type %T", value)
	}
	if len(data) == 0 {
		*tiers = nil
		return nil
	}
	return common.Unmarshal(data, tiers)
}

type ProfitCostModelGroup struct {
	Group  string   `json:"group"`
	Models []string `json:"models"`
}

func (rule *ModelCostRule) AfterFind(*gorm.DB) error {
	if rule.PurchasePriceCNY == 0 && rule.UpstreamPriceUSD > 0 && rule.ExchangeRate > 0 && rule.Discount > 0 {
		rule.PurchasePriceCNY = rule.UpstreamPriceUSD * rule.ExchangeRate * rule.Discount
	}
	return nil
}

type ProfitAnalysisState struct {
	Id         int   `gorm:"primaryKey"`
	Generation int64 `gorm:"not null;default:0"`
	ResetAt    int64 `gorm:"index"`
}

type ProfitResetLogKey struct {
	Generation   int64  `gorm:"primaryKey;autoIncrement:false"`
	SourceLogKey string `gorm:"type:varchar(128);primaryKey;autoIncrement:false"`
}

type ProfitRecord struct {
	Id                      int64  `json:"id" gorm:"primaryKey"`
	SourceLogKey            string `json:"source_log_key" gorm:"type:varchar(128);uniqueIndex:idx_profit_source_generation,priority:1"`
	Generation              int64  `json:"-" gorm:"not null;default:0;uniqueIndex:idx_profit_source_generation,priority:2;index"`
	SourceRequestId         string `json:"source_request_id" gorm:"type:varchar(128);index"`
	OccurredAt              int64  `json:"occurred_at" gorm:"index"`
	UserId                  int    `json:"user_id" gorm:"index"`
	ModelName               string `json:"model_name" gorm:"type:varchar(191);index"`
	CostModelName           string `json:"cost_model_name" gorm:"type:varchar(191);index"`
	ChannelId               int    `json:"channel_id" gorm:"index"`
	Group                   string `json:"group" gorm:"type:varchar(64);index"`
	GrossConsumptionMicros  int64  `json:"gross_consumption_micros"`
	RevenueMicros           int64  `json:"revenue_micros"`
	PromoConsumptionMicros  int64  `json:"promo_consumption_micros"`
	LegacyConsumptionMicros int64  `json:"legacy_consumption_micros"`
	AdminConsumptionMicros  int64  `json:"admin_consumption_micros"`
	PromoCostMicros         int64  `json:"promo_cost_micros"`
	AdminCostMicros         int64  `json:"admin_cost_micros"`
	RevenueClass            string `json:"revenue_class" gorm:"type:varchar(32);index"`
	UserRoleSnapshot        int    `json:"user_role_snapshot"`
	AttributionKnown        bool   `json:"attribution_known" gorm:"index"`
	AttributionVersion      int    `json:"-" gorm:"not null;default:0;index"`
	CostMicros              int64  `json:"cost_micros"`
	CostKnown               bool   `json:"cost_known" gorm:"index"`
	Estimated               bool   `json:"estimated" gorm:"index"`
	CostRuleId              int64  `json:"cost_rule_id" gorm:"index"`
	CostRuleVersion         int    `json:"cost_rule_version"`
	BillingSnapshot         string `json:"billing_snapshot" gorm:"type:text"`
}

type ProfitQuery struct {
	StartTimestamp int64
	EndTimestamp   int64
	UserId         int
	ModelName      string
	ChannelId      int
	Group          string
	UserIds        []int
	Generation     *int64
}

type ProfitAggregate struct {
	UserId                   int      `json:"user_id,omitempty"`
	Username                 string   `json:"username,omitempty" gorm:"-"`
	ModelName                string   `json:"model_name,omitempty"`
	ChannelId                int      `json:"channel_id,omitempty"`
	ChannelName              string   `json:"channel_name,omitempty" gorm:"-"`
	Group                    string   `json:"group,omitempty"`
	GrossConsumptionMicros   int64    `json:"gross_consumption_micros"`
	RevenueMicros            int64    `json:"revenue_micros"`
	KnownRevenueMicros       int64    `json:"known_revenue_micros"`
	PromoConsumptionMicros   int64    `json:"promo_consumption_micros"`
	LegacyConsumptionMicros  int64    `json:"legacy_consumption_micros"`
	AdminConsumptionMicros   int64    `json:"admin_consumption_micros"`
	PromoCostMicros          int64    `json:"promo_cost_micros"`
	AdminCostMicros          int64    `json:"admin_cost_micros"`
	PromoUnpricedRecordCount int64    `json:"promo_unpriced_record_count"`
	AdminUnpricedRecordCount int64    `json:"admin_unpriced_record_count"`
	CostMicros               int64    `json:"cost_micros"`
	ProfitMicros             int64    `json:"profit_micros"`
	RecordCount              int64    `json:"record_count"`
	UnpricedRecordCount      int64    `json:"unpriced_record_count"`
	ProfitMargin             *float64 `json:"profit_margin" gorm:"-"`
	CostCoverage             float64  `json:"cost_coverage" gorm:"-"`
}

func (record *ProfitRecord) ProfitMicros() int64 {
	if record == nil || !record.CostKnown {
		return 0
	}
	return record.RevenueMicros - record.CostMicros
}

type profitBillingSnapshot struct {
	ModelPrice        float64 `json:"model_price"`
	ModelRatio        float64 `json:"model_ratio"`
	GroupRatio        float64 `json:"group_ratio"`
	UpstreamModelName string  `json:"upstream_model_name"`
	ProfitGeneration  *int64  `json:"profit_generation"`
	BillingSource     string  `json:"billing_source"`
	IsAdminUsage      bool    `json:"is_admin_usage"`
	UserRoleSnapshot  int     `json:"user_role_snapshot"`
	CostTier          string  `json:"cost_tier"`
	OtherMultiplier   float64 `json:"other_multiplier"`
	WalletFunding     struct {
		Version     int `json:"version"`
		PaidQuota   int `json:"paid_quota"`
		PromoQuota  int `json:"promo_quota"`
		LegacyQuota int `json:"legacy_quota"`
	} `json:"wallet_funding"`
}

func attachProfitGeneration(log *Log, generation int64) (int64, error) {
	if log == nil {
		return generation, errors.New("log is required")
	}
	other := make(map[string]interface{})
	if log.Other != "" {
		if err := common.UnmarshalJsonStr(log.Other, &other); err != nil {
			return generation, err
		}
		var snapshot profitBillingSnapshot
		if err := common.UnmarshalJsonStr(log.Other, &snapshot); err != nil {
			return generation, err
		}
		if snapshot.ProfitGeneration != nil {
			if *snapshot.ProfitGeneration < 0 {
				return generation, errors.New("profit generation must not be negative")
			}
			return *snapshot.ProfitGeneration, nil
		}
	}
	other["profit_generation"] = generation
	encoded, err := common.Marshal(other)
	if err != nil {
		return generation, err
	}
	log.Other = string(encoded)
	return generation, nil
}

func profitSourceLogKey(log *Log) string {
	if log.Id > 0 {
		return fmt.Sprintf("id:%d", log.Id)
	}
	payload := fmt.Sprintf("%d\x00%d\x00%d\x00%s\x00%s\x00%d\x00%d\x00%d\x00%s\x00%s\x00%s",
		log.CreatedAt, log.Type, log.UserId, log.RequestId, log.ModelName, log.Quota,
		log.ChannelId, log.TokenId, log.Group, log.UpstreamRequestId, log.Other)
	sum := sha256.Sum256([]byte(payload))
	return "hash:" + hex.EncodeToString(sum[:])
}

func uniqueProfitSourceLogKeys(logs []*Log) []string {
	keySet := make(map[string]struct{}, len(logs))
	for _, log := range logs {
		keySet[profitSourceLogKey(log)] = struct{}{}
	}
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func profitCostModelName(log *Log) string {
	if log == nil || log.Other == "" {
		if log == nil {
			return ""
		}
		return log.ModelName
	}
	var snapshot profitBillingSnapshot
	if common.UnmarshalJsonStr(log.Other, &snapshot) == nil && strings.TrimSpace(snapshot.UpstreamModelName) != "" {
		return strings.TrimSpace(snapshot.UpstreamModelName)
	}
	return log.ModelName
}

func validateModelCostRule(rule *ModelCostRule) error {
	if rule == nil {
		return errors.New("cost rule is required")
	}
	rule.ModelName = strings.TrimSpace(rule.ModelName)
	if rule.ModelName == "" || len(rule.ModelName) > 191 {
		return errors.New("model name is required and must not exceed 191 characters")
	}
	if math.IsNaN(rule.PurchasePriceCNY) || math.IsInf(rule.PurchasePriceCNY, 0) {
		return errors.New("purchase price must be finite")
	}
	if rule.PurchasePriceCNY <= 0 || rule.PurchasePriceCNY > 1_000_000 {
		return errors.New("purchase price must be greater than 0 and at most 1000000")
	}
	if len(rule.CostTiers) > 64 {
		return errors.New("cost tiers must not exceed 64 entries")
	}
	tierKeys := make(map[string]struct{}, len(rule.CostTiers))
	for index := range rule.CostTiers {
		tier := &rule.CostTiers[index]
		tier.Key = strings.TrimSpace(tier.Key)
		tier.Label = strings.TrimSpace(tier.Label)
		if tier.Key == "" || len(tier.Key) > 128 {
			return errors.New("cost tier key is required and must not exceed 128 characters")
		}
		if tier.Label == "" || len(tier.Label) > 191 {
			return errors.New("cost tier label is required and must not exceed 191 characters")
		}
		if _, exists := tierKeys[tier.Key]; exists {
			return errors.New("cost tier keys must be unique")
		}
		tierKeys[tier.Key] = struct{}{}
		if math.IsNaN(tier.PurchasePriceCNY) || math.IsInf(tier.PurchasePriceCNY, 0) ||
			tier.PurchasePriceCNY <= 0 || tier.PurchasePriceCNY > 1_000_000 {
			return errors.New("cost tier purchase price must be finite, greater than 0, and at most 1000000")
		}
	}
	return nil
}

func calculateProfitRecord(log *Log, rule *ModelCostRule, estimated bool) (*ProfitRecord, error) {
	if log == nil {
		return nil, errors.New("log is required")
	}
	if log.Type != LogTypeConsume && log.Type != LogTypeRefund {
		return nil, errors.New("profit records only support consume and refund logs")
	}
	if common.QuotaPerUnit <= 0 {
		return nil, errors.New("quota per unit must be greater than 0")
	}
	sign := int64(1)
	if log.Type == LogTypeRefund {
		sign = -1
	}
	grossConsumptionMicros := decimal.NewFromInt(int64(log.Quota)).
		Mul(decimal.NewFromInt(profitMicrosPerUnit)).
		Div(decimal.NewFromFloat(common.QuotaPerUnit)).
		Round(0).
		IntPart() * sign
	revenueMicros := int64(0)
	var snapshot profitBillingSnapshot
	attributionKnown := false
	revenueClass := "legacy_unknown"
	promoConsumptionMicros := int64(0)
	legacyConsumptionMicros := grossConsumptionMicros
	adminConsumptionMicros := int64(0)
	if log.Other != "" && common.UnmarshalJsonStr(log.Other, &snapshot) == nil {
		if snapshot.WalletFunding.Version > 0 {
			attributionKnown = true
			legacyConsumptionMicros = 0
			quotaMicros := func(quota int) int64 {
				return decimal.NewFromInt(int64(quota)).
					Mul(decimal.NewFromInt(profitMicrosPerUnit)).
					Div(decimal.NewFromFloat(common.QuotaPerUnit)).
					Round(0).IntPart() * sign
			}
			revenueMicros = quotaMicros(snapshot.WalletFunding.PaidQuota)
			promoConsumptionMicros = quotaMicros(snapshot.WalletFunding.PromoQuota)
			legacyConsumptionMicros = quotaMicros(snapshot.WalletFunding.LegacyQuota)
			switch {
			case snapshot.WalletFunding.PaidQuota > 0 && (snapshot.WalletFunding.PromoQuota > 0 || snapshot.WalletFunding.LegacyQuota > 0):
				revenueClass = "mixed"
			case snapshot.WalletFunding.PaidQuota > 0:
				revenueClass = "paid"
			case snapshot.WalletFunding.PromoQuota > 0:
				revenueClass = "promo"
			default:
				revenueClass = "legacy_unknown"
			}
		}
		if snapshot.IsAdminUsage {
			revenueMicros = 0
			promoConsumptionMicros = 0
			legacyConsumptionMicros = 0
			adminConsumptionMicros = grossConsumptionMicros
			revenueClass = "admin"
			attributionKnown = true
		}
	}
	record := &ProfitRecord{
		SourceLogKey:            profitSourceLogKey(log),
		SourceRequestId:         log.RequestId,
		OccurredAt:              log.CreatedAt,
		UserId:                  log.UserId,
		ModelName:               log.ModelName,
		CostModelName:           profitCostModelName(log),
		ChannelId:               log.ChannelId,
		Group:                   log.Group,
		GrossConsumptionMicros:  grossConsumptionMicros,
		RevenueMicros:           revenueMicros,
		PromoConsumptionMicros:  promoConsumptionMicros,
		LegacyConsumptionMicros: legacyConsumptionMicros,
		AdminConsumptionMicros:  adminConsumptionMicros,
		RevenueClass:            revenueClass,
		UserRoleSnapshot:        snapshot.UserRoleSnapshot,
		AttributionKnown:        attributionKnown,
		AttributionVersion:      currentProfitAttributionVersion,
		Estimated:               estimated,
		BillingSnapshot:         log.Other,
	}
	if rule == nil {
		return record, nil
	}
	if err := validateModelCostRule(rule); err != nil {
		return nil, err
	}
	if log.Other == "" || common.UnmarshalJsonStr(log.Other, &snapshot) != nil {
		return record, nil
	}
	saleBase := 0.0
	if snapshot.ModelPrice > 0 {
		saleBase = snapshot.ModelPrice * snapshot.GroupRatio
	} else if snapshot.ModelRatio > 0 {
		saleBase = snapshot.ModelRatio * 2 * snapshot.GroupRatio
	}
	if saleBase <= 0 || math.IsNaN(saleBase) || math.IsInf(saleBase, 0) {
		return record, nil
	}
	costBase := rule.PurchasePriceCNY
	actualSaleBase := saleBase
	if snapshot.CostTier != "" {
		matchedCostTier := false
		for _, tier := range rule.CostTiers {
			if tier.Key == snapshot.CostTier {
				costBase = tier.PurchasePriceCNY
				matchedCostTier = true
				break
			}
		}
		if matchedCostTier && snapshot.OtherMultiplier > 0 && !math.IsNaN(snapshot.OtherMultiplier) && !math.IsInf(snapshot.OtherMultiplier, 0) {
			actualSaleBase *= snapshot.OtherMultiplier
		}
	}
	if actualSaleBase <= 0 || math.IsNaN(actualSaleBase) || math.IsInf(actualSaleBase, 0) {
		return record, nil
	}
	costMicros := decimal.NewFromInt(grossConsumptionMicros).
		Mul(decimal.NewFromFloat(costBase)).
		Div(decimal.NewFromFloat(actualSaleBase)).
		Round(0).
		IntPart()
	record.CostMicros = costMicros
	if adminConsumptionMicros != 0 {
		record.AdminCostMicros = costMicros
	} else if promoConsumptionMicros != 0 && grossConsumptionMicros != 0 {
		record.PromoCostMicros = decimal.NewFromInt(costMicros).
			Mul(decimal.NewFromInt(promoConsumptionMicros)).
			Div(decimal.NewFromInt(grossConsumptionMicros)).
			Round(0).IntPart()
	}
	record.CostKnown = true
	record.CostRuleId = rule.Id
	record.CostRuleVersion = rule.Version
	return record, nil
}

func GetModelCostRules() ([]*ModelCostRule, error) {
	rules := make([]*ModelCostRule, 0)
	err := DB.Order("model_name ASC, version DESC").Find(&rules).Error
	return rules, err
}

func getConfiguredProfitCostModelNames() ([]string, error) {
	modelNames := make(map[string]struct{})
	sources := []struct {
		model  interface{}
		column string
		where  string
		args   []interface{}
	}{
		{model: &ProfitRecord{}, column: "cost_model_name", where: "cost_model_name <> ?", args: []interface{}{string("")}},
		{model: &ModelCostRule{}, column: "model_name", where: "model_name <> ?", args: []interface{}{string("")}},
	}
	for _, source := range sources {
		var names []string
		if err := DB.Model(source.model).Where(source.where, source.args...).Distinct(source.column).Pluck(source.column, &names).Error; err != nil {
			return nil, err
		}
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				modelNames[name] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(modelNames))
	for name := range modelNames {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func GetProfitCostModelGroups() ([]ProfitCostModelGroup, error) {
	type abilityModel struct {
		Group string `gorm:"column:model_group"`
		Model string `gorm:"column:model"`
	}

	groupedModels := make(map[string]map[string]struct{})
	modelSeenInGroup := make(map[string]struct{})
	addModel := func(group, modelName string) {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			return
		}
		if _, ok := groupedModels[group]; !ok {
			groupedModels[group] = make(map[string]struct{})
		}
		groupedModels[group][modelName] = struct{}{}
		if group != "" {
			modelSeenInGroup[modelName] = struct{}{}
		}
	}

	abilityModels := make([]abilityModel, 0)
	if err := DB.Model(&Ability{}).
		Select(commonGroupCol+" AS model_group, model").
		Where("enabled = ?", true).
		Group(commonGroupCol + ", model").
		Order(commonGroupCol + " ASC, model ASC").
		Scan(&abilityModels).Error; err != nil {
		return nil, err
	}
	for _, row := range abilityModels {
		addModel(row.Group, row.Model)
	}

	configuredNames, err := getConfiguredProfitCostModelNames()
	if err != nil {
		return nil, err
	}
	for _, modelName := range configuredNames {
		if _, ok := modelSeenInGroup[modelName]; !ok {
			addModel("", modelName)
		}
	}

	groups := make([]string, 0, len(groupedModels))
	for group := range groupedModels {
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i] == "" {
			return false
		}
		if groups[j] == "" {
			return true
		}
		return groups[i] < groups[j]
	})

	result := make([]ProfitCostModelGroup, 0, len(groups))
	for _, group := range groups {
		models := make([]string, 0, len(groupedModels[group]))
		for modelName := range groupedModels[group] {
			models = append(models, modelName)
		}
		sort.Strings(models)
		result = append(result, ProfitCostModelGroup{Group: group, Models: models})
	}
	return result, nil
}

func GetProfitCostModelNames() ([]string, error) {
	groups, err := GetProfitCostModelGroups()
	if err != nil {
		return nil, err
	}
	modelNames := make(map[string]struct{})
	for _, group := range groups {
		for _, modelName := range group.Models {
			modelNames[modelName] = struct{}{}
		}
	}
	result := make([]string, 0, len(modelNames))
	for modelName := range modelNames {
		result = append(result, modelName)
	}
	sort.Strings(result)
	return result, nil
}

func GetActiveModelCostRule(modelName string, occurredAt int64) (*ModelCostRule, error) {
	if DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var rule ModelCostRule
	tx := DB.Where("model_name = ? AND effective_from <= ? AND (effective_to = 0 OR effective_to > ?)",
		modelName, occurredAt, occurredAt).
		Order("version DESC").
		Limit(1).Find(&rule)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rule, nil
}

func getCurrentModelCostRule(modelName string) (*ModelCostRule, error) {
	var rule ModelCostRule
	tx := DB.Where("model_name = ? AND enabled = ?", modelName, true).
		Order("version DESC").Limit(1).Find(&rule)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rule, nil
}

func SaveModelCostRule(input *ModelCostRule) (*ModelCostRule, error) {
	if err := validateModelCostRule(input); err != nil {
		return nil, err
	}
	modelNames, err := GetProfitCostModelNames()
	if err != nil {
		return nil, err
	}
	modelIndex := sort.SearchStrings(modelNames, input.ModelName)
	if modelIndex >= len(modelNames) || modelNames[modelIndex] != input.ModelName {
		return nil, errors.New("model name is not available for profit cost configuration")
	}
	now := time.Now().Unix()
	rule := &ModelCostRule{
		ModelName:        input.ModelName,
		PurchasePriceCNY: input.PurchasePriceCNY,
		CostTiers:        append(ModelCostTiers(nil), input.CostTiers...),
		Enabled:          true,
		EffectiveFrom:    now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		var maxVersion int
		if err := tx.Model(&ModelCostRule{}).
			Where("model_name = ?", rule.ModelName).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion).Error; err != nil {
			return err
		}
		rule.Version = maxVersion + 1
		if err := tx.Model(&ModelCostRule{}).
			Where("model_name = ? AND enabled = ?", rule.ModelName, true).
			Updates(map[string]interface{}{"enabled": false, "effective_to": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Create(rule).Error
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func recordProfitForLog(log *Log, estimated bool, generation int64) error {
	if DB == nil || log == nil || (log.Type != LogTypeConsume && log.Type != LogTypeRefund) {
		return nil
	}
	rule, err := GetActiveModelCostRule(profitCostModelName(log), log.CreatedAt)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rule = nil
	} else if err != nil {
		return err
	}
	record, err := calculateProfitRecord(log, rule, estimated)
	if err != nil {
		return err
	}
	record.Generation = generation
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_log_key"}, {Name: "generation"}},
		DoNothing: true,
	}).Create(record).Error
}

func BackfillProfitRecords() error {
	if !profitBackfillMutex.TryLock() {
		return ErrProfitBackfillInProgress
	}
	defer profitBackfillMutex.Unlock()
	if DB == nil || LOG_DB == nil {
		return nil
	}
	const batchSize = 500
	lastID := 0
	offset := 0
	useOffset := common.UsingLogDatabase(common.DatabaseTypeClickHouse)
	ruleCache := make(map[string]*ModelCostRule)
	missingRules := make(map[string]bool)
	resetLogKeyGeneration := int64(-1)
	resetLogKeys := make(map[string]bool)
	scanGeneration := int64(-1)
	for {
		state, err := getProfitAnalysisState(DB)
		if err != nil {
			return err
		}
		if scanGeneration != state.Generation {
			lastID = 0
			offset = 0
			ruleCache = make(map[string]*ModelCostRule)
			missingRules = make(map[string]bool)
			scanGeneration = state.Generation
		}
		logs := make([]*Log, 0, batchSize)
		query := LOG_DB.Where("type IN ?", []int{LogTypeConsume, LogTypeRefund})
		if state.ResetAt > 0 {
			query = query.Where("created_at >= ?", state.ResetAt)
		}
		if resetLogKeyGeneration != state.Generation {
			var keys []string
			if err := DB.Model(&ProfitResetLogKey{}).Where("generation = ?", state.Generation).Pluck("source_log_key", &keys).Error; err != nil {
				return err
			}
			resetLogKeys = make(map[string]bool, len(keys))
			for _, key := range keys {
				resetLogKeys[key] = true
			}
			resetLogKeyGeneration = state.Generation
		}
		if useOffset {
			query = query.Order("created_at ASC, request_id ASC, type ASC, user_id ASC, model_name ASC, quota ASC, channel_id ASC, token_id ASC, other ASC").Offset(offset)
		} else {
			query = query.Where("id > ?", lastID).Order("id ASC")
		}
		err = query.Limit(batchSize).Find(&logs).Error
		if err != nil {
			return err
		}
		if len(logs) == 0 {
			latestState, stateErr := getProfitAnalysisState(DB)
			if stateErr != nil {
				return stateErr
			}
			if latestState.Generation != scanGeneration {
				continue
			}
			return nil
		}
		for _, log := range logs {
			lastID = log.Id
			logKey := profitSourceLogKey(log)
			var snapshot profitBillingSnapshot
			if common.UnmarshalJsonStr(log.Other, &snapshot) == nil && snapshot.ProfitGeneration != nil && *snapshot.ProfitGeneration != state.Generation {
				continue
			}
			if log.CreatedAt == state.ResetAt && resetLogKeys[logKey] {
				continue
			}
			var existing ProfitRecord
			existingQuery := DB.Where("source_log_key = ? AND generation = ?", logKey, state.Generation).Limit(1).Find(&existing)
			if existingQuery.Error != nil {
				return existingQuery.Error
			}
			exists := existingQuery.RowsAffected > 0
			if exists && existing.CostKnown && existing.AttributionVersion >= currentProfitAttributionVersion {
				continue
			}
			costModelName := profitCostModelName(log)
			rule := ruleCache[costModelName]
			if rule == nil && !missingRules[costModelName] {
				var ruleErr error
				rule, ruleErr = getCurrentModelCostRule(costModelName)
				if errors.Is(ruleErr, gorm.ErrRecordNotFound) {
					missingRules[costModelName] = true
					rule = nil
				} else if ruleErr != nil {
					return ruleErr
				} else {
					ruleCache[costModelName] = rule
				}
			}
			record, calcErr := calculateProfitRecord(log, rule, true)
			if calcErr != nil {
				return calcErr
			}
			record.Generation = state.Generation
			if exists {
				record.Id = existing.Id
				if updateErr := DB.Save(record).Error; updateErr != nil {
					return updateErr
				}
				continue
			}
			if createErr := DB.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "source_log_key"}, {Name: "generation"}},
				DoNothing: true,
			}).Create(record).Error; createErr != nil {
				return createErr
			}
		}
		if useOffset {
			offset += len(logs)
		}
	}
}

func ResetProfitAnalysisData(resetAt int64) error {
	if resetAt <= 0 {
		return errors.New("reset timestamp must be greater than 0")
	}
	if !profitBackfillMutex.TryLock() {
		return ErrProfitBackfillInProgress
	}
	defer profitBackfillMutex.Unlock()
	profitEpochMutex.Lock()
	defer profitEpochMutex.Unlock()
	resetLogKeys := make([]string, 0)
	if LOG_DB != nil {
		logs := make([]*Log, 0)
		if err := LOG_DB.Where("type IN ? AND created_at = ?", []int{LogTypeConsume, LogTypeRefund}, resetAt).Find(&logs).Error; err != nil {
			return err
		}
		resetLogKeys = uniqueProfitSourceLogKeys(logs)
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&ProfitAnalysisState{Id: 1}).Error; err != nil {
			return err
		}
		var state ProfitAnalysisState
		if err := lockForUpdate(tx).Where("id = ?", 1).First(&state).Error; err != nil {
			return err
		}
		state.Generation++
		state.ResetAt = resetAt
		if err := tx.Model(&ProfitAnalysisState{}).Where("id = ?", state.Id).Updates(map[string]interface{}{
			"generation": state.Generation,
			"reset_at":   state.ResetAt,
		}).Error; err != nil {
			return err
		}
		if len(resetLogKeys) > 0 {
			rows := make([]*ProfitResetLogKey, 0, len(resetLogKeys))
			for _, sourceLogKey := range resetLogKeys {
				rows = append(rows, &ProfitResetLogKey{Generation: state.Generation, SourceLogKey: sourceLogKey})
			}
			if err := tx.CreateInBatches(rows, 500).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func getProfitAnalysisState(db *gorm.DB) (*ProfitAnalysisState, error) {
	state := &ProfitAnalysisState{Id: 1}
	query := db.Where("id = ?", state.Id).Limit(1).Find(state)
	if query.Error != nil {
		return nil, query.Error
	}
	return state, nil
}

func GetProfitAnalysisGeneration() (int64, error) {
	state, err := getProfitAnalysisState(DB)
	if err != nil {
		return 0, err
	}
	return state.Generation, nil
}

func applyProfitQuery(tx *gorm.DB, query ProfitQuery) *gorm.DB {
	tx = tx.Where("generation = ?", *query.Generation)
	if query.StartTimestamp > 0 {
		tx = tx.Where("occurred_at >= ?", query.StartTimestamp)
	}
	if query.EndTimestamp > 0 {
		tx = tx.Where("occurred_at <= ?", query.EndTimestamp)
	}
	if query.UserId > 0 {
		tx = tx.Where("user_id = ?", query.UserId)
	}
	if len(query.UserIds) > 0 {
		tx = tx.Where("user_id IN ?", query.UserIds)
	}
	if query.ModelName != "" {
		tx = tx.Where("model_name = ?", query.ModelName)
	}
	if query.ChannelId > 0 {
		tx = tx.Where("channel_id = ?", query.ChannelId)
	}
	if query.Group != "" {
		tx = tx.Where(commonGroupCol+" = ?", query.Group)
	}
	return tx
}

func finalizeProfitAggregate(row *ProfitAggregate) {
	if row == nil {
		return
	}
	row.ProfitMicros = row.KnownRevenueMicros - row.CostMicros
	if row.KnownRevenueMicros != 0 {
		margin := float64(row.ProfitMicros) / float64(row.KnownRevenueMicros)
		row.ProfitMargin = &margin
	}
	pricedCount := row.RecordCount - row.UnpricedRecordCount
	if row.RecordCount > 0 {
		row.CostCoverage = float64(pricedCount) / float64(row.RecordCount)
	}
}

func profitAggregateSelect(columns string) string {
	prefix := ""
	if columns != "" {
		prefix = columns + ", "
	}
	return prefix +
		"COALESCE(SUM(gross_consumption_micros), 0) AS gross_consumption_micros, " +
		"COALESCE(SUM(revenue_micros), 0) AS revenue_micros, " +
		"COALESCE(SUM(CASE WHEN cost_known = ? THEN revenue_micros ELSE 0 END), 0) AS known_revenue_micros, " +
		"COALESCE(SUM(promo_consumption_micros), 0) AS promo_consumption_micros, " +
		"COALESCE(SUM(legacy_consumption_micros), 0) AS legacy_consumption_micros, " +
		"COALESCE(SUM(admin_consumption_micros), 0) AS admin_consumption_micros, " +
		"COALESCE(SUM(promo_cost_micros), 0) AS promo_cost_micros, " +
		"COALESCE(SUM(admin_cost_micros), 0) AS admin_cost_micros, " +
		"COALESCE(SUM(CASE WHEN promo_consumption_micros <> 0 AND cost_known = ? THEN 1 ELSE 0 END), 0) AS promo_unpriced_record_count, " +
		"COALESCE(SUM(CASE WHEN admin_consumption_micros <> 0 AND cost_known = ? THEN 1 ELSE 0 END), 0) AS admin_unpriced_record_count, " +
		"COALESCE(SUM(cost_micros), 0) AS cost_micros, " +
		"COALESCE(SUM(CASE WHEN cost_known = ? THEN revenue_micros ELSE 0 END), 0) - COALESCE(SUM(cost_micros), 0) AS profit_micros, " +
		"COUNT(*) AS record_count, " +
		"COALESCE(SUM(CASE WHEN cost_known = ? THEN 0 ELSE 1 END), 0) AS unpriced_record_count"
}

func GetProfitAggregate(query ProfitQuery) (*ProfitAggregate, error) {
	if query.Generation == nil {
		generation, err := GetProfitAnalysisGeneration()
		if err != nil {
			return nil, err
		}
		query.Generation = &generation
	}
	row := &ProfitAggregate{}
	tx := applyProfitQuery(DB.Model(&ProfitRecord{}), query)
	err := tx.Select(profitAggregateSelect(""), true, false, false, true, true).Scan(row).Error
	if err != nil {
		return nil, err
	}
	finalizeProfitAggregate(row)
	return row, nil
}

func GetProfitBreakdown(query ProfitQuery, dimension string) ([]*ProfitAggregate, error) {
	columnByDimension := map[string]string{
		"user":    "user_id",
		"model":   "model_name",
		"channel": "channel_id",
		"group":   commonGroupCol,
	}
	column, ok := columnByDimension[dimension]
	if !ok {
		return nil, errors.New("invalid profit breakdown dimension")
	}
	if query.Generation == nil {
		generation, err := GetProfitAnalysisGeneration()
		if err != nil {
			return nil, err
		}
		query.Generation = &generation
	}
	rows := make([]*ProfitAggregate, 0)
	tx := applyProfitQuery(DB.Model(&ProfitRecord{}), query)
	if err := tx.Select(profitAggregateSelect(column), true, false, false, true, true).
		Group(column).
		Order("profit_micros DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		finalizeProfitAggregate(row)
	}
	return rows, nil
}

func GetUserProfitSummaries(userIds []int) (map[int]*ProfitAggregate, error) {
	result := make(map[int]*ProfitAggregate, len(userIds))
	if len(userIds) == 0 {
		return result, nil
	}
	rows, err := GetProfitBreakdown(ProfitQuery{UserIds: userIds}, "user")
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UserId] = row
	}
	return result, nil
}
