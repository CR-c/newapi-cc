package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResetProfitAnalysisDataStartsNewGenerationAndPreservesRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalDB, originalLogDB := model.DB, model.LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ModelCostRule{}, &model.ProfitRecord{}, &model.ProfitAnalysisState{}, &model.ProfitResetLogKey{}))
	model.DB = db
	model.LOG_DB = nil
	t.Cleanup(func() { model.DB, model.LOG_DB = originalDB, originalLogDB })

	require.NoError(t, db.Create(&model.ModelCostRule{
		ModelName: "video", PurchasePriceCNY: 1, Version: 1, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&model.ProfitRecord{SourceLogKey: "old", ModelName: "video"}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/profit/reset", nil)
	ResetProfitAnalysisData(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)

	var recordCount, ruleCount int64
	require.NoError(t, db.Model(&model.ProfitRecord{}).Count(&recordCount).Error)
	require.NoError(t, db.Model(&model.ModelCostRule{}).Count(&ruleCount).Error)
	assert.Equal(t, int64(1), recordCount)
	assert.Equal(t, int64(1), ruleCount)
	summary, err := model.GetProfitAggregate(model.ProfitQuery{})
	require.NoError(t, err)
	assert.Zero(t, summary.RecordCount)
}
