package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachGroupModelPricesOnlyIncludesUsableGroups(t *testing.T) {
	original := ratio_setting.GroupModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupModelPriceByJSONString(original))
	})
	require.NoError(t, ratio_setting.UpdateGroupModelPriceByJSONString(`{
		"enterprise": {"video": 5},
		"private": {"video": 3}
	}`))

	pricing := []model.Pricing{{
		ModelName:   "video",
		QuotaType:   1,
		EnableGroup: []string{"enterprise", "private"},
	}}
	result := attachGroupModelPrices(pricing, map[string]string{"enterprise": "Enterprise"})

	require.Len(t, result, 1)
	assert.Equal(t, map[string]float64{"enterprise": 5}, result[0].GroupModelPrice)
	assert.Nil(t, pricing[0].GroupModelPrice)
}

func TestAttachGroupModelPricesSkipsTieredModels(t *testing.T) {
	original := ratio_setting.GroupModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupModelPriceByJSONString(original))
	})
	require.NoError(t, ratio_setting.UpdateGroupModelPriceByJSONString(`{
		"enterprise": {"tiered": 5}
	}`))

	result := attachGroupModelPrices([]model.Pricing{{
		ModelName:   "tiered",
		QuotaType:   1,
		BillingMode: "tiered_expr",
		EnableGroup: []string{"enterprise"},
	}}, map[string]string{"enterprise": "Enterprise"})

	require.Len(t, result, 1)
	assert.Nil(t, result[0].GroupModelPrice)
}
