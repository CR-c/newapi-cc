package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGroupRealSalesRatioFallsBackToBillingRatio(t *testing.T) {
	require.NoError(t, UpdateGroupRealSalesRatioByJSONString(`{}`))
	t.Cleanup(func() {
		_ = UpdateGroupRealSalesRatioByJSONString(`{}`)
	})

	assert.Equal(t, 7.5, ResolveGroupRealSalesRatio("Corn专用", 7.5))
	assert.Equal(t, 1.0, ResolveGroupRealSalesRatio("", 0))
}

func TestResolveGroupRealSalesRatioUsesConfiguredOverride(t *testing.T) {
	require.NoError(t, UpdateGroupRealSalesRatioByJSONString(`{"Corn专用":6.3}`))
	t.Cleanup(func() {
		_ = UpdateGroupRealSalesRatioByJSONString(`{}`)
	})

	ratio, ok := GetGroupRealSalesRatio("Corn专用")
	require.True(t, ok)
	assert.Equal(t, 6.3, ratio)
	assert.Equal(t, 6.3, ResolveGroupRealSalesRatio("Corn专用", 7.5))

	_, ok = GetGroupRealSalesRatio("video-dddd")
	assert.False(t, ok)
	assert.Equal(t, 6.3, ResolveGroupRealSalesRatio("video-dddd", 6.3))
}

func TestUpdateGroupRealSalesRatioRejectsNegative(t *testing.T) {
	err := UpdateGroupRealSalesRatioByJSONString(`{"Corn专用":-1}`)
	require.Error(t, err)
}
