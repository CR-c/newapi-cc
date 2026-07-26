package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateTaskPrechargeQuotaUsesEstimatedUsage(t *testing.T) {
	priceData := types.PriceData{
		ModelRatio: 1.75,
		GroupRatioInfo: types.GroupRatioInfo{
			GroupRatio: 6.3,
		},
	}
	priceData.AddOtherRatio("video_input", 2.1/3.5)

	quota, clamp, ok := calculateTaskPrechargeQuota(priceData, 432900)

	require.True(t, ok)
	assert.Nil(t, clamp)
	assert.Equal(t, 2863633, quota)
}

func TestCalculateTaskPrechargeQuotaSkipsFixedPriceModels(t *testing.T) {
	quota, clamp, ok := calculateTaskPrechargeQuota(types.PriceData{
		UsePrice:   true,
		ModelPrice: 3,
	}, 196344)

	assert.False(t, ok)
	assert.Nil(t, clamp)
	assert.Zero(t, quota)
}
