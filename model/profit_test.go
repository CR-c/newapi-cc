package model

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type clickHouseProfitLogFixture struct {
	LogID     int    `gorm:"column:id;default:0"`
	CreatedAt int64  `gorm:"column:created_at"`
	Type      int    `gorm:"column:type"`
	UserId    int    `gorm:"column:user_id"`
	RequestId string `gorm:"column:request_id"`
	ModelName string `gorm:"column:model_name"`
	Quota     int    `gorm:"column:quota"`
	ChannelId int    `gorm:"column:channel_id"`
	TokenId   int    `gorm:"column:token_id"`
	Group     string `gorm:"column:group"`
	Other     string `gorm:"column:other"`
}

func (clickHouseProfitLogFixture) TableName() string { return "logs" }

type legacyProfitRecordFixture struct {
	Id           int64  `gorm:"primaryKey"`
	SourceLogKey string `gorm:"type:varchar(128);uniqueIndex"`
}

func (legacyProfitRecordFixture) TableName() string { return "profit_records" }

type legacyProfitAnalysisStateFixture struct {
	Id      int   `gorm:"primaryKey"`
	ResetAt int64 `gorm:"index"`
}

func (legacyProfitAnalysisStateFixture) TableName() string { return "profit_analysis_states" }

func TestCalculateProfitRecordForFixedPriceConsume(t *testing.T) {
	log := &Log{
		RequestId: "req-fixed",
		CreatedAt: 100,
		Type:      LogTypeConsume,
		UserId:    7,
		ModelName: "image-model",
		Quota:     50000,
		Other:     `{"model_price":0.1,"group_ratio":1}`,
	}
	rule := &ModelCostRule{
		Id:               3,
		ModelName:        "image-model",
		PurchasePriceCNY: 0.06,
		Version:          2,
	}

	record, err := calculateProfitRecord(log, rule, false)
	require.NoError(t, err)
	assert.Zero(t, record.RevenueMicros)
	assert.Equal(t, int64(100000), record.LegacyConsumptionMicros)
	assert.Equal(t, int64(60000), record.CostMicros)
	assert.True(t, record.CostKnown)
	assert.Equal(t, int64(-60000), record.ProfitMicros())
	assert.Equal(t, int64(3), record.CostRuleId)
	assert.Equal(t, 2, record.CostRuleVersion)
}

func TestCalculateProfitRecordUsesFundingAttributionForRecognizedRevenue(t *testing.T) {
	tests := []struct {
		name            string
		other           string
		expectedRevenue int64
		expectedPromo   int64
		expectedLegacy  int64
		expectedAdmin   int64
		expectedCost    int64
		expectedProfit  int64
	}{
		{
			name:            "paid wallet usage",
			other:           `{"model_price":0.1,"group_ratio":1,"wallet_funding":{"version":1,"paid_quota":50000}}`,
			expectedRevenue: 100000,
			expectedCost:    60000,
			expectedProfit:  40000,
		},
		{
			name:           "promotional wallet usage",
			other:          `{"model_price":0.1,"group_ratio":1,"wallet_funding":{"version":1,"promo_quota":50000}}`,
			expectedPromo:  100000,
			expectedCost:   60000,
			expectedProfit: -60000,
		},
		{
			name:           "promotional subscription usage",
			other:          `{"model_price":0.1,"group_ratio":1,"billing_source":"subscription","wallet_funding":{"version":1,"promo_quota":50000}}`,
			expectedPromo:  100000,
			expectedCost:   60000,
			expectedProfit: -60000,
		},
		{
			name:           "legacy unknown wallet usage",
			other:          `{"model_price":0.1,"group_ratio":1,"wallet_funding":{"version":1,"legacy_quota":50000}}`,
			expectedLegacy: 100000,
			expectedCost:   60000,
			expectedProfit: -60000,
		},
		{
			name:           "administrator paid usage",
			other:          `{"model_price":0.1,"group_ratio":1,"is_admin_usage":true,"user_role_snapshot":10,"wallet_funding":{"version":1,"paid_quota":50000}}`,
			expectedAdmin:  100000,
			expectedCost:   60000,
			expectedProfit: -60000,
		},
	}

	rule := &ModelCostRule{Id: 3, ModelName: "image-model", PurchasePriceCNY: 0.06, Version: 2}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record, err := calculateProfitRecord(&Log{
				RequestId: "req-attribution", CreatedAt: 100, Type: LogTypeConsume,
				UserId: 7, ModelName: "image-model", Quota: 50000, Other: test.other,
			}, rule, false)
			require.NoError(t, err)
			assert.Equal(t, int64(100000), record.GrossConsumptionMicros)
			assert.Equal(t, test.expectedRevenue, record.RevenueMicros)
			assert.Equal(t, test.expectedPromo, record.PromoConsumptionMicros)
			assert.Equal(t, test.expectedLegacy, record.LegacyConsumptionMicros)
			assert.Equal(t, test.expectedAdmin, record.AdminConsumptionMicros)
			assert.Equal(t, test.expectedCost, record.CostMicros)
			assert.Equal(t, test.expectedProfit, record.ProfitMicros())
		})
	}
}

func TestGetProfitAggregateNetsRefundsAndExcludesUnpricedRevenueFromMargin(t *testing.T) {
	originalDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ProfitRecord{}, &ProfitAnalysisState{}))
	DB = db
	t.Cleanup(func() { DB = originalDB })

	require.NoError(t, db.Create([]*ProfitRecord{
		{SourceLogKey: "consume", SourceRequestId: "consume", UserId: 7, ModelName: "priced", RevenueMicros: 1_000_000, CostMicros: 600_000, CostKnown: true},
		{SourceLogKey: "refund", SourceRequestId: "refund", UserId: 7, ModelName: "priced", RevenueMicros: -200_000, CostMicros: -120_000, CostKnown: true},
		{SourceLogKey: "unknown", SourceRequestId: "unknown", UserId: 7, ModelName: "unknown", RevenueMicros: 500_000, CostKnown: false},
	}).Error)

	summary, err := GetProfitAggregate(ProfitQuery{UserId: 7})
	require.NoError(t, err)
	assert.Equal(t, int64(1_300_000), summary.RevenueMicros)
	assert.Equal(t, int64(800_000), summary.KnownRevenueMicros)
	assert.Equal(t, int64(480_000), summary.CostMicros)
	assert.Equal(t, int64(320_000), summary.ProfitMicros)
	require.NotNil(t, summary.ProfitMargin)
	assert.InDelta(t, 0.4, *summary.ProfitMargin, 0.000001)
	assert.InDelta(t, 2.0/3.0, summary.CostCoverage, 0.000001)
	assert.Equal(t, int64(1), summary.UnpricedRecordCount)
}

func TestGetProfitAggregateReportsUnpricedAttributionCosts(t *testing.T) {
	originalDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ProfitRecord{}, &ProfitAnalysisState{}))
	DB = db
	t.Cleanup(func() { DB = originalDB })

	require.NoError(t, db.Create([]*ProfitRecord{
		{SourceLogKey: "promo-unpriced", PromoConsumptionMicros: 100_000, CostKnown: false},
		{SourceLogKey: "promo-priced", PromoConsumptionMicros: 200_000, CostMicros: 50_000, CostKnown: true},
		{SourceLogKey: "admin-unpriced", AdminConsumptionMicros: 300_000, CostKnown: false},
	}).Error)

	summary, err := GetProfitAggregate(ProfitQuery{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), summary.PromoUnpricedRecordCount)
	assert.Equal(t, int64(1), summary.AdminUnpricedRecordCount)
}

func TestProfitSourceLogKeyDistinguishesBillingEventsWithSameRequestId(t *testing.T) {
	consume := &Log{RequestId: "same-request", CreatedAt: 100, Type: LogTypeConsume, Quota: 500}
	refund := &Log{RequestId: "same-request", CreatedAt: 100, Type: LogTypeRefund, Quota: 200}

	assert.NotEqual(t, profitSourceLogKey(consume), profitSourceLogKey(refund))
}

func TestUniqueProfitSourceLogKeysDeduplicatesClickHouseRows(t *testing.T) {
	duplicate := &Log{RequestId: "same", CreatedAt: 100, Type: LogTypeConsume, Quota: 500}
	different := &Log{RequestId: "different", CreatedAt: 100, Type: LogTypeConsume, Quota: 500}

	keys := uniqueProfitSourceLogKeys([]*Log{duplicate, duplicate, different})
	assert.Len(t, keys, 2)
	assert.Less(t, keys[0], keys[1])
}

func TestGetActiveModelCostRuleFindsHistoricalDisabledVersion(t *testing.T) {
	originalDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}))
	DB = db
	t.Cleanup(func() { DB = originalDB })

	require.NoError(t, db.Create([]*ModelCostRule{
		{ModelName: "video", Version: 1, Enabled: false, EffectiveFrom: 100, EffectiveTo: 200},
		{ModelName: "video", Version: 2, Enabled: true, EffectiveFrom: 200},
	}).Error)

	v1, err := GetActiveModelCostRule("video", 150)
	require.NoError(t, err)
	assert.Equal(t, 1, v1.Version)
	v2, err := GetActiveModelCostRule("video", 250)
	require.NoError(t, err)
	assert.Equal(t, 2, v2.Version)
}

func TestCalculateProfitRecordUsesMappedUpstreamModel(t *testing.T) {
	log := &Log{
		RequestId: "alias-request", Type: LogTypeConsume, ModelName: "public-alias", Quota: 50000,
		Other: `{"model_price":0.1,"group_ratio":1,"upstream_model_name":"dreamina-seedance-2-0-hc"}`,
	}
	rule := &ModelCostRule{ModelName: "dreamina-seedance-2-0-hc", PurchasePriceCNY: 0.06}

	record, err := calculateProfitRecord(log, rule, false)
	require.NoError(t, err)
	assert.Equal(t, "public-alias", record.ModelName)
	assert.Equal(t, "dreamina-seedance-2-0-hc", record.CostModelName)
	assert.True(t, record.CostKnown)
}

func TestResolveProfitCostRulePrefersOriginModelOverUpstream(t *testing.T) {
	originalDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}))
	DB = db
	t.Cleanup(func() { DB = originalDB })

	now := time.Now().Unix()
	require.NoError(t, db.Create([]*ModelCostRule{
		{ModelName: "gpt-image-2", PurchasePriceCNY: 0.06, Version: 1, Enabled: true, EffectiveFrom: now - 10},
		{ModelName: "gpt-image-2-ic", PurchasePriceCNY: 0.01, Version: 1, Enabled: true, EffectiveFrom: now - 10},
	}).Error)

	log := &Log{
		RequestId: "ic-mapped",
		Type:      LogTypeConsume,
		ModelName: "gpt-image-2-ic",
		CreatedAt: now,
		Quota:     10000,
		Other:     `{"model_price":0.02,"group_ratio":1,"upstream_model_name":"gpt-image-2"}`,
	}
	rule, costModelName, err := resolveProfitCostRule(log, func(modelName string) (*ModelCostRule, error) {
		return GetActiveModelCostRule(modelName, log.CreatedAt)
	})
	require.NoError(t, err)
	require.NotNil(t, rule)
	assert.Equal(t, "gpt-image-2-ic", costModelName)
	assert.Equal(t, 0.01, rule.PurchasePriceCNY)

	record, err := calculateProfitRecord(log, rule, false)
	require.NoError(t, err)
	assert.Equal(t, "gpt-image-2-ic", record.CostModelName)
	assert.True(t, record.CostKnown)
	// Quota 10000 with QuotaPerUnit 500000 → gross 0.02 units.
	// sale 0.02, cost 0.01 → cost micros = gross * 0.01/0.02
	assert.Equal(t, int64(20000), record.GrossConsumptionMicros)
	assert.Equal(t, int64(10000), record.CostMicros)
}

func TestResolveProfitCostRuleFallsBackToUpstreamWhenOriginMissing(t *testing.T) {
	originalDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}))
	DB = db
	t.Cleanup(func() { DB = originalDB })

	now := time.Now().Unix()
	require.NoError(t, db.Create(&ModelCostRule{
		ModelName: "dreamina-seedance-2-0-hc", PurchasePriceCNY: 0.06, Version: 1, Enabled: true, EffectiveFrom: now - 10,
	}).Error)

	log := &Log{
		RequestId: "alias-only-upstream-cost",
		Type:      LogTypeConsume,
		ModelName: "public-alias",
		CreatedAt: now,
		Quota:     50000,
		Other:     `{"model_price":0.1,"group_ratio":1,"upstream_model_name":"dreamina-seedance-2-0-hc"}`,
	}
	rule, costModelName, err := resolveProfitCostRule(log, func(modelName string) (*ModelCostRule, error) {
		return GetActiveModelCostRule(modelName, log.CreatedAt)
	})
	require.NoError(t, err)
	require.NotNil(t, rule)
	assert.Equal(t, "dreamina-seedance-2-0-hc", costModelName)
	assert.Equal(t, 0.06, rule.PurchasePriceCNY)
}

func TestCreateBillingLogRecordsProfitWhenConsumeLogsAreDisabled(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	originalLogConsumeEnabled := common.LogConsumeEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}, &ProfitRecord{}, &ProfitAnalysisState{}, &ProfitResetLogKey{}))
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.Exec(`CREATE TABLE logs (
		id INTEGER DEFAULT 0, created_at INTEGER, type INTEGER, user_id INTEGER,
		request_id TEXT, model_name TEXT, quota INTEGER, channel_id INTEGER,
		token_id INTEGER, "group" TEXT, other TEXT
	)`).Error)
	DB, LOG_DB = db, logDB
	common.LogConsumeEnabled = false
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.LogConsumeEnabled = originalLogConsumeEnabled
	})

	require.NoError(t, createBillingLog(&Log{
		CreatedAt: 100, Type: LogTypeConsume, UserId: 9, ModelName: "unpriced", Quota: 100,
	}))

	var logCount, profitCount int64
	require.NoError(t, logDB.Model(&Log{}).Count(&logCount).Error)
	require.NoError(t, db.Model(&ProfitRecord{}).Count(&profitCount).Error)
	assert.Zero(t, logCount)
	assert.Equal(t, int64(1), profitCount)
}

func TestCreateLogPersistsProfitGenerationForBackfill(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	originalLogConsumeEnabled := common.LogConsumeEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}, &ProfitRecord{}, &ProfitAnalysisState{}))
	require.NoError(t, db.Create(&ProfitAnalysisState{Id: 1, Generation: 3}).Error)
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.AutoMigrate(&Log{}))
	DB, LOG_DB = db, logDB
	common.LogConsumeEnabled = true
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.LogConsumeEnabled = originalLogConsumeEnabled
	})

	require.NoError(t, createBillingLog(&Log{
		CreatedAt: 100, Type: LogTypeConsume, UserId: 1, ModelName: "video", Quota: 100,
		Other: `{"model_price":1,"group_ratio":1}`,
	}))

	var storedLog Log
	require.NoError(t, logDB.First(&storedLog).Error)
	var snapshot profitBillingSnapshot
	require.NoError(t, common.UnmarshalJsonStr(storedLog.Other, &snapshot))
	require.NotNil(t, snapshot.ProfitGeneration)
	assert.Equal(t, int64(3), *snapshot.ProfitGeneration)
	var record ProfitRecord
	require.NoError(t, db.First(&record).Error)
	assert.Equal(t, int64(3), record.Generation)
}

func TestCreateLogPreservesExplicitProfitGenerationAcrossReset(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	originalLogConsumeEnabled := common.LogConsumeEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}, &ProfitRecord{}, &ProfitAnalysisState{}))
	require.NoError(t, db.Create(&ProfitAnalysisState{Id: 1, Generation: 2}).Error)
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.AutoMigrate(&Log{}))
	DB, LOG_DB = db, logDB
	common.LogConsumeEnabled = true
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.LogConsumeEnabled = originalLogConsumeEnabled
	})

	require.NoError(t, createBillingLog(&Log{
		CreatedAt: 100, Type: LogTypeRefund, UserId: 1, ModelName: "video", Quota: 100,
		Other: `{"model_price":1,"group_ratio":1,"profit_generation":1}`,
	}))

	var storedLog Log
	require.NoError(t, logDB.First(&storedLog).Error)
	var snapshot profitBillingSnapshot
	require.NoError(t, common.UnmarshalJsonStr(storedLog.Other, &snapshot))
	require.NotNil(t, snapshot.ProfitGeneration)
	assert.Equal(t, int64(1), *snapshot.ProfitGeneration)
	var record ProfitRecord
	require.NoError(t, db.First(&record).Error)
	assert.Equal(t, int64(1), record.Generation)
}

func TestAsyncBillingEventsStayTogetherAcrossProfitReset(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	originalLogConsumeEnabled := common.LogConsumeEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}, &ProfitRecord{}, &ProfitAnalysisState{}, &ProfitResetLogKey{}))
	require.NoError(t, db.Create(&ProfitAnalysisState{Id: 1, Generation: 1}).Error)
	require.NoError(t, db.Create(&ModelCostRule{
		ModelName: "video", PurchasePriceCNY: 0.6, Version: 1, Enabled: true,
	}).Error)
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.AutoMigrate(&Log{}))
	DB, LOG_DB = db, logDB
	common.LogConsumeEnabled = true
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.LogConsumeEnabled = originalLogConsumeEnabled
	})

	require.NoError(t, createBillingLog(&Log{
		CreatedAt: 100, Type: LogTypeConsume, UserId: 1, ModelName: "video", Quota: 500_000,
		Other: `{"model_price":1,"group_ratio":1,"profit_generation":1}`,
	}))
	require.NoError(t, ResetProfitAnalysisData(200))
	require.NoError(t, createBillingLog(&Log{
		CreatedAt: 300, Type: LogTypeRefund, UserId: 1, ModelName: "video", Quota: 200_000,
		Other: `{"model_price":1,"group_ratio":1,"profit_generation":1}`,
	}))

	current, err := GetProfitAggregate(ProfitQuery{UserId: 1})
	require.NoError(t, err)
	assert.Zero(t, current.RecordCount)
	previousGeneration := int64(1)
	previous, err := GetProfitAggregate(ProfitQuery{UserId: 1, Generation: &previousGeneration})
	require.NoError(t, err)
	assert.Equal(t, int64(2), previous.RecordCount)
	assert.Zero(t, previous.RevenueMicros)
	assert.Equal(t, int64(600_000), previous.LegacyConsumptionMicros)
	assert.Equal(t, int64(360_000), previous.CostMicros)
	assert.Equal(t, int64(-360_000), previous.ProfitMicros)
}

func TestBackfillProfitRecordsClickHouseModeUsesOffsetAndIsIdempotent(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	originalLogDatabaseType := common.LogDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}, &ProfitRecord{}, &ProfitAnalysisState{}, &ProfitResetLogKey{}))
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.Exec(`CREATE TABLE logs (
		id INTEGER DEFAULT 0, created_at INTEGER, type INTEGER, user_id INTEGER,
		request_id TEXT, model_name TEXT, quota INTEGER, channel_id INTEGER,
		token_id INTEGER, "group" TEXT, other TEXT
	)`).Error)
	DB, LOG_DB = db, logDB
	common.SetLogDatabaseType(common.DatabaseTypeClickHouse)
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.SetLogDatabaseType(originalLogDatabaseType)
	})

	logs := make([]*clickHouseProfitLogFixture, 0, 501)
	for i := 0; i < 501; i++ {
		logs = append(logs, &clickHouseProfitLogFixture{
			CreatedAt: 100, Type: LogTypeConsume, UserId: 1, RequestId: "shared-request",
			ModelName: "unpriced", Quota: i + 1, Other: fmt.Sprintf(`{"sequence":%d}`, i),
		})
	}
	require.NoError(t, logDB.CreateInBatches(logs, 100).Error)

	require.NoError(t, BackfillProfitRecords())
	require.NoError(t, BackfillProfitRecords())
	var count int64
	require.NoError(t, db.Model(&ProfitRecord{}).Count(&count).Error)
	assert.Equal(t, int64(501), count)
}

func TestBackfillProfitRecordsRecalculatesLegacyAttributionVersion(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	originalLogDatabaseType := common.LogDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}, &ProfitRecord{}, &ProfitAnalysisState{}, &ProfitResetLogKey{}))
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.Exec(`CREATE TABLE logs (
		id INTEGER PRIMARY KEY, created_at INTEGER, type INTEGER, user_id INTEGER,
		request_id TEXT, model_name TEXT, quota INTEGER, channel_id INTEGER,
		token_id INTEGER, "group" TEXT, other TEXT
	)`).Error)
	DB, LOG_DB = db, logDB
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.SetLogDatabaseType(originalLogDatabaseType)
	})

	require.NoError(t, db.Create(&ModelCostRule{
		ModelName: "video", PurchasePriceCNY: 0.6, Version: 1, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&ProfitRecord{
		SourceLogKey: "id:1", Generation: 0, ModelName: "video",
		RevenueMicros: 1_000_000, CostMicros: 600_000, CostKnown: true,
	}).Error)
	require.NoError(t, logDB.Create(&clickHouseProfitLogFixture{
		LogID: 1, CreatedAt: 100, Type: LogTypeConsume, RequestId: "legacy-attribution",
		ModelName: "video", Quota: 500_000,
		Other: `{"model_price":1,"group_ratio":1,"wallet_funding":{"version":1,"promo_quota":500000}}`,
	}).Error)

	require.NoError(t, BackfillProfitRecords())
	var record ProfitRecord
	require.NoError(t, db.Where("source_log_key = ?", "id:1").First(&record).Error)
	assert.Equal(t, currentProfitAttributionVersion, record.AttributionVersion)
	assert.Zero(t, record.RevenueMicros)
	assert.Equal(t, int64(1_000_000), record.PromoConsumptionMicros)
	assert.Equal(t, int64(600_000), record.CostMicros)
}

func TestBackfillProfitRecordsRestartsWhenGenerationChanges(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	originalLogDatabaseType := common.LogDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}, &ProfitRecord{}, &ProfitAnalysisState{}, &ProfitResetLogKey{}))
	require.NoError(t, db.Create(&ProfitAnalysisState{Id: 1}).Error)
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.Exec(`CREATE TABLE logs (
		id INTEGER PRIMARY KEY, created_at INTEGER, type INTEGER, user_id INTEGER,
		request_id TEXT, model_name TEXT, quota INTEGER, channel_id INTEGER,
		token_id INTEGER, "group" TEXT, other TEXT
	)`).Error)
	DB, LOG_DB = db, logDB
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.SetLogDatabaseType(originalLogDatabaseType)
	})

	logs := make([]*clickHouseProfitLogFixture, 0, 501)
	for i := 1; i <= 501; i++ {
		logs = append(logs, &clickHouseProfitLogFixture{
			LogID: i, CreatedAt: 100, Type: LogTypeConsume, UserId: 1,
			RequestId: fmt.Sprintf("generation-one-%d", i), ModelName: "unpriced",
			Quota: i, Other: `{"profit_generation":1}`,
		})
	}
	require.NoError(t, logDB.CreateInBatches(logs, 100).Error)

	advancedGeneration := false
	var advanceErr error
	require.NoError(t, logDB.Callback().Query().Before("gorm:query").Register("test:advance_profit_generation", func(*gorm.DB) {
		if advancedGeneration {
			return
		}
		advancedGeneration = true
		advanceErr = db.Model(&ProfitAnalysisState{}).Where("id = ?", 1).Update("generation", 1).Error
	}))
	t.Cleanup(func() {
		require.NoError(t, logDB.Callback().Query().Remove("test:advance_profit_generation"))
	})

	require.NoError(t, BackfillProfitRecords())
	require.NoError(t, advanceErr)
	var count int64
	require.NoError(t, db.Model(&ProfitRecord{}).Where("generation = ?", 1).Count(&count).Error)
	assert.Equal(t, int64(501), count)
}

func TestCalculateProfitRecordRefundReversesRevenueAndCost(t *testing.T) {
	log := &Log{
		RequestId: "req-refund",
		Type:      LogTypeRefund,
		ModelName: "image-model",
		Quota:     50000,
		Other:     `{"model_price":0.1,"group_ratio":1}`,
	}
	rule := &ModelCostRule{
		ModelName:        "image-model",
		PurchasePriceCNY: 0.06,
	}

	record, err := calculateProfitRecord(log, rule, false)
	require.NoError(t, err)
	assert.Zero(t, record.RevenueMicros)
	assert.Equal(t, int64(-100000), record.LegacyConsumptionMicros)
	assert.Equal(t, int64(-60000), record.CostMicros)
	assert.Equal(t, int64(60000), record.ProfitMicros())
}

func TestCalculateProfitRecordScalesRevenueByGroupRealSalesRatio(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	// Corn-style reseller markup: billed at 7.5, platform real sales 6.3.
	require.NoError(t, ratio_setting.UpdateGroupRealSalesRatioByJSONString(`{"Corn专用":6.3}`))
	t.Cleanup(func() {
		_ = ratio_setting.UpdateGroupRealSalesRatioByJSONString(`{}`)
	})

	rule := &ModelCostRule{Id: 9, ModelName: "image-model", PurchasePriceCNY: 0.06, Version: 1}
	record, err := calculateProfitRecord(&Log{
		RequestId: "req-real-sales",
		CreatedAt: 100,
		Type:      LogTypeConsume,
		UserId:    7,
		ModelName: "image-model",
		Group:     "Corn专用",
		Quota:     50000,
		Other:     `{"model_price":0.1,"group_ratio":7.5,"wallet_funding":{"version":1,"paid_quota":50000}}`,
	}, rule, false)
	require.NoError(t, err)

	// Gross stays full billed amount; revenue scaled by 6.3/7.5.
	assert.Equal(t, int64(100000), record.GrossConsumptionMicros)
	assert.Equal(t, int64(84000), record.RevenueMicros)
	// cost = gross * 0.06 / (0.1 * 7.5) = 8000
	assert.Equal(t, int64(8000), record.CostMicros)
	assert.Equal(t, int64(76000), record.ProfitMicros())

	// Unset group keeps full recognized revenue.
	recordDefault, err := calculateProfitRecord(&Log{
		RequestId: "req-default-group",
		CreatedAt: 100,
		Type:      LogTypeConsume,
		UserId:    7,
		ModelName: "image-model",
		Group:     "video-dddd",
		Quota:     50000,
		Other:     `{"model_price":0.1,"group_ratio":6.3,"wallet_funding":{"version":1,"paid_quota":50000}}`,
	}, rule, false)
	require.NoError(t, err)
	assert.Equal(t, int64(100000), recordDefault.RevenueMicros)
}

func TestCalculateProfitRecordForRatioModel(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	log := &Log{
		RequestId: "req-ratio",
		Type:      LogTypeConsume,
		ModelName: "video-token-model",
		Quota:     341145,
		Other:     `{"model_price":-1,"model_ratio":1.75,"group_ratio":1}`,
	}
	rule := &ModelCostRule{
		ModelName:        "video-token-model",
		PurchasePriceCNY: 2.1,
	}

	record, err := calculateProfitRecord(log, rule, false)
	require.NoError(t, err)
	assert.Zero(t, record.RevenueMicros)
	assert.Equal(t, int64(682290), record.LegacyConsumptionMicros)
	assert.Equal(t, int64(409374), record.CostMicros)
	assert.Equal(t, int64(-409374), record.ProfitMicros())
}

func TestCalculateProfitRecordUsesTierPurchasePriceAndActualSaleBase(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	log := &Log{
		RequestId: "req-video-tier",
		Type:      LogTypeConsume,
		ModelName: "doubao-seedance-2-0-260128",
		Quota:     11200000,
		Other: `{"model_price":-1,"model_ratio":23,"group_ratio":0.8,` +
			`"cost_tier":"video:480p-720p:reference","other_multiplier":0.6086956521739131}`,
	}
	rule := &ModelCostRule{
		ModelName:        "doubao-seedance-2-0-260128",
		PurchasePriceCNY: 29.9,
		CostTiers: ModelCostTiers{
			{Key: "video:480p-720p:reference", Label: "480p/720p with video", PurchasePriceCNY: 18.2},
		},
	}

	record, err := calculateProfitRecord(log, rule, false)
	require.NoError(t, err)
	assert.True(t, record.CostKnown)
	assert.Equal(t, int64(18_200_000), record.CostMicros)
}

func TestCalculateProfitRecordFallsBackToBasePurchasePriceForUnknownTier(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	log := &Log{
		RequestId: "req-video-tier-fallback",
		Type:      LogTypeConsume,
		ModelName: "video-model",
		Quota:     500000,
		Other:     `{"model_price":1,"group_ratio":1,"cost_tier":"video:unknown","other_multiplier":0.5}`,
	}
	rule := &ModelCostRule{
		ModelName:        "video-model",
		PurchasePriceCNY: 0.6,
		CostTiers: ModelCostTiers{
			{Key: "video:known", Label: "Known", PurchasePriceCNY: 0.4},
		},
	}

	record, err := calculateProfitRecord(log, rule, false)
	require.NoError(t, err)
	assert.Equal(t, int64(600_000), record.CostMicros)
}

func TestValidateModelCostRuleRejectsInvalidCostTiers(t *testing.T) {
	tests := []struct {
		name  string
		tiers ModelCostTiers
	}{
		{name: "empty key", tiers: ModelCostTiers{{Label: "Base", PurchasePriceCNY: 1}}},
		{name: "empty label", tiers: ModelCostTiers{{Key: "video:base", PurchasePriceCNY: 1}}},
		{name: "duplicate key", tiers: ModelCostTiers{
			{Key: "video:base", Label: "Base", PurchasePriceCNY: 1},
			{Key: " video:base ", Label: "Duplicate", PurchasePriceCNY: 2},
		}},
		{name: "invalid price", tiers: ModelCostTiers{{Key: "video:base", Label: "Base", PurchasePriceCNY: 0}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rule := &ModelCostRule{ModelName: "video-model", PurchasePriceCNY: 1, CostTiers: test.tiers}
			require.Error(t, validateModelCostRule(rule))
		})
	}
}

func TestDreaminaCostVariantsUseConfiguredTierPurchasePrice(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	tests := []struct {
		name             string
		model            string
		baseSalePrice    float64
		purchasePriceCNY float64
		variantSalePrice float64
		expectedCostCNY  float64
		costTier         string
	}{
		{"hc-4k-no-ref", "dreamina-seedance-2-0-hc", 7, 38.08, 4, 21.76, "video:4k:no-reference"},
		{"hc-4k-with-ref", "dreamina-seedance-2-0-hc", 7, 38.08, 2.4, 13.056, "video:4k:reference"},
		{"hc-480p-no-ref", "dreamina-seedance-2-0-hc", 7, 38.08, 7, 38.08, "video:480p-720p:no-reference"},
		{"hc-480p-with-ref", "dreamina-seedance-2-0-hc", 7, 38.08, 4.3, 23.392, "video:480p-720p:reference"},
		{"hc-720p-no-ref", "dreamina-seedance-2-0-hc", 7, 38.08, 7, 38.08, "video:480p-720p:no-reference"},
		{"hc-720p-with-ref", "dreamina-seedance-2-0-hc", 7, 38.08, 4.3, 23.392, "video:480p-720p:reference"},
		{"hc-1080p-no-ref", "dreamina-seedance-2-0-hc", 7, 38.08, 7.7, 41.888, "video:1080p:no-reference"},
		{"hc-1080p-with-ref", "dreamina-seedance-2-0-hc", 7, 38.08, 4.7, 25.568, "video:1080p:reference"},
		{"fast-no-ref", "dreamina-seedance-2-0-fast-hc", 5.6, 30.464, 5.6, 30.464, "video:480p-720p:no-reference"},
		{"fast-with-ref", "dreamina-seedance-2-0-fast-hc", 5.6, 30.464, 3.3, 17.952, "video:480p-720p:reference"},
		{"mini-no-ref", "dreamina-seedance-2-0-mini-hc", 3.5, 19.04, 3.5, 19.04, "video:480p-720p:no-reference"},
		{"mini-with-ref", "dreamina-seedance-2-0-mini-hc", 3.5, 19.04, 2.1, 11.424, "video:480p-720p:reference"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			quota := int(math.Round(500000 * test.variantSalePrice))
			otherMultiplier := test.variantSalePrice / test.baseSalePrice
			log := &Log{
				RequestId: test.name, Type: LogTypeConsume, ModelName: test.model, Quota: quota,
				Other: fmt.Sprintf(`{"model_price":-1,"model_ratio":%g,"group_ratio":1,"cost_tier":%q,"other_multiplier":%g}`,
					test.baseSalePrice/2, test.costTier, otherMultiplier),
			}
			rule := &ModelCostRule{
				ModelName: test.model, PurchasePriceCNY: test.purchasePriceCNY,
				CostTiers: ModelCostTiers{{Key: test.costTier, Label: test.name, PurchasePriceCNY: test.expectedCostCNY}},
			}

			record, err := calculateProfitRecord(log, rule, false)
			require.NoError(t, err)
			expectedCostMicros := int64(math.Round(test.expectedCostCNY * 1_000_000))
			assert.Equal(t, expectedCostMicros, record.CostMicros)
		})
	}
}

func TestResetProfitAnalysisDataPreventsHistoricalBackfill(t *testing.T) {
	originalDB, originalLogDB := DB, LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ModelCostRule{}, &ProfitRecord{}, &ProfitAnalysisState{}, &ProfitResetLogKey{}, &Ability{}))
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.Exec(`CREATE TABLE logs (
		id INTEGER PRIMARY KEY, created_at INTEGER, type INTEGER, user_id INTEGER,
		request_id TEXT, model_name TEXT, quota INTEGER, channel_id INTEGER,
		token_id INTEGER, "group" TEXT, other TEXT
	)`).Error)
	DB, LOG_DB = db, logDB
	t.Cleanup(func() { DB, LOG_DB = originalDB, originalLogDB })

	require.NoError(t, db.Create(&ModelCostRule{
		ModelName: "video", PurchasePriceCNY: 0.6, Version: 1, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&ProfitRecord{
		SourceLogKey: "old", OccurredAt: 100, ModelName: "video", RevenueMicros: 1_000_000,
	}).Error)
	require.NoError(t, logDB.Create(&clickHouseProfitLogFixture{
		LogID: 1, CreatedAt: 100, Type: LogTypeConsume, RequestId: "old", ModelName: "video",
		Quota: 500_000, Other: `{"model_price":1,"group_ratio":1}`,
	}).Error)
	require.NoError(t, logDB.Create(&clickHouseProfitLogFixture{
		LogID: 2, CreatedAt: 200, Type: LogTypeConsume, RequestId: "old-same-second", ModelName: "video",
		Quota: 500_000, Other: `{"model_price":1,"group_ratio":1}`,
	}).Error)

	require.NoError(t, ResetProfitAnalysisData(200))
	require.NoError(t, BackfillProfitRecords())

	var recordCount, ruleCount int64
	require.NoError(t, db.Model(&ProfitRecord{}).Count(&recordCount).Error)
	require.NoError(t, db.Model(&ModelCostRule{}).Count(&ruleCount).Error)
	assert.Equal(t, int64(1), recordCount)
	assert.Equal(t, int64(1), ruleCount)
	require.NoError(t, db.Create(&ProfitRecord{
		SourceLogKey: "stale", Generation: 0, OccurredAt: 100, RevenueMicros: 9_000_000,
	}).Error)
	require.NoError(t, recordProfitForLog(&Log{
		CreatedAt: 200, Type: LogTypeConsume, RequestId: "same-second", ModelName: "video",
		Quota: 500_000, Other: `{"model_price":1,"group_ratio":1}`,
	}, false, 1))

	summary, err := GetProfitAggregate(ProfitQuery{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), summary.RecordCount)
	assert.Zero(t, summary.RevenueMicros)
	assert.Equal(t, int64(1_000_000), summary.LegacyConsumptionMicros)

	require.NoError(t, logDB.Create(&clickHouseProfitLogFixture{
		LogID: 3, CreatedAt: 200, Type: LogTypeConsume, RequestId: "new-same-second", ModelName: "video",
		Quota: 500_000, Other: `{"model_price":1,"group_ratio":1}`,
	}).Error)
	require.NoError(t, BackfillProfitRecords())
	summary, err = GetProfitAggregate(ProfitQuery{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), summary.RecordCount)

	require.NoError(t, logDB.Create(&clickHouseProfitLogFixture{
		LogID: 4, CreatedAt: 200, Type: LogTypeConsume, RequestId: "late-old-generation", ModelName: "video",
		Quota: 500_000, Other: `{"model_price":1,"group_ratio":1,"profit_generation":0}`,
	}).Error)
	require.NoError(t, BackfillProfitRecords())
	summary, err = GetProfitAggregate(ProfitQuery{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), summary.RecordCount)

	require.NoError(t, logDB.Create(&clickHouseProfitLogFixture{
		LogID: 5, CreatedAt: 201, Type: LogTypeConsume, RequestId: "new", ModelName: "video",
		Quota: 500_000, Other: `{"model_price":1,"group_ratio":1}`,
	}).Error)
	require.NoError(t, BackfillProfitRecords())
	summary, err = GetProfitAggregate(ProfitQuery{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), summary.RecordCount)
}

func TestGetProfitCostModelNamesReturnsStableUnion(t *testing.T) {
	originalDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ProfitRecord{}, &ModelCostRule{}, &Ability{}))
	DB = db
	t.Cleanup(func() { DB = originalDB })

	require.NoError(t, db.Create(&ProfitRecord{SourceLogKey: "mapped", CostModelName: "upstream-model"}).Error)
	require.NoError(t, db.Create(&ModelCostRule{ModelName: "legacy-model", Version: 1}).Error)
	require.NoError(t, db.Create(&Ability{Group: "kyy-sd", Model: "sd-model", ChannelId: 1, Enabled: true}).Error)

	names, err := GetProfitCostModelNames()
	require.NoError(t, err)
	assert.Equal(t, []string{"legacy-model", "sd-model", "upstream-model"}, names)

	groups, err := GetProfitCostModelGroups()
	require.NoError(t, err)
	assert.Equal(t, []ProfitCostModelGroup{
		{Group: "kyy-sd", Models: []string{"sd-model"}},
		{Group: "", Models: []string{"legacy-model", "upstream-model"}},
	}, groups)

	savedRule, err := SaveModelCostRule(&ModelCostRule{
		ModelName: "sd-model", PurchasePriceCNY: 1,
		CostTiers: ModelCostTiers{{Key: "video:base", Label: "Base", PurchasePriceCNY: 0.65}},
	})
	assert.NoError(t, err)
	require.NotNil(t, savedRule)
	var persistedRule ModelCostRule
	require.NoError(t, db.First(&persistedRule, savedRule.Id).Error)
	assert.Equal(t, savedRule.CostTiers, persistedRule.CostTiers)

	_, err = SaveModelCostRule(&ModelCostRule{ModelName: "not-in-list", PurchasePriceCNY: 1})
	assert.ErrorContains(t, err, "model name is not available")
}

func TestMigrateProfitRecordIndexesReplacesLegacyUniqueIndex(t *testing.T) {
	originalDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyProfitRecordFixture{}))
	require.NoError(t, db.Create(&legacyProfitRecordFixture{SourceLogKey: "legacy"}).Error)
	require.True(t, db.Migrator().HasIndex(&ProfitRecord{}, "idx_profit_records_source_log_key"))
	require.NoError(t, db.AutoMigrate(&ProfitRecord{}))
	DB = db
	t.Cleanup(func() { DB = originalDB })

	require.NoError(t, migrateProfitRecordIndexes())
	assert.False(t, db.Migrator().HasIndex(&ProfitRecord{}, "idx_profit_records_source_log_key"))
	assert.True(t, db.Migrator().HasIndex(&ProfitRecord{}, "idx_profit_source_generation"))
	var migrated ProfitRecord
	require.NoError(t, db.Where("source_log_key = ?", "legacy").First(&migrated).Error)
	assert.Zero(t, migrated.Generation)
}

func TestProfitAnalysisStateMigrationDefaultsLegacyGeneration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyProfitAnalysisStateFixture{}))
	require.NoError(t, db.Create(&legacyProfitAnalysisStateFixture{Id: 1, ResetAt: 100}).Error)

	require.NoError(t, db.AutoMigrate(&ProfitAnalysisState{}))
	var state ProfitAnalysisState
	require.NoError(t, db.First(&state, 1).Error)
	assert.Zero(t, state.Generation)
	assert.Equal(t, int64(100), state.ResetAt)
}

func TestCalculateProfitRecordWithoutCostRuleStaysUnpriced(t *testing.T) {
	log := &Log{
		RequestId: "req-unpriced",
		Type:      LogTypeConsume,
		ModelName: "unknown",
		Quota:     50000,
	}

	record, err := calculateProfitRecord(log, nil, true)
	require.NoError(t, err)
	assert.Zero(t, record.RevenueMicros)
	assert.Equal(t, int64(100000), record.LegacyConsumptionMicros)
	assert.Zero(t, record.CostMicros)
	assert.False(t, record.CostKnown)
	assert.True(t, record.Estimated)
}
