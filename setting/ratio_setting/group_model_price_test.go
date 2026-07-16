package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupModelPrice(t *testing.T) {
	original := GroupModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupModelPriceByJSONString(original))
	})

	require.NoError(t, UpdateGroupModelPriceByJSONString(`{
		"enterprise": {
			"video-fast": 5,
			"video-pro": 6
		}
	}`))

	price, ok := GetGroupModelPrice("enterprise", "video-fast")
	require.True(t, ok)
	assert.Equal(t, 5.0, price)

	_, ok = GetGroupModelPrice("enterprise", "unknown")
	assert.False(t, ok)
	_, ok = GetGroupModelPrice("default", "video-fast")
	assert.False(t, ok)
}

func TestUpdateGroupModelPriceRejectsInvalidPrices(t *testing.T) {
	original := GroupModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupModelPriceByJSONString(original))
	})

	require.Error(t, UpdateGroupModelPriceByJSONString(`{"enterprise":{"video":-1}}`))
	require.Error(t, UpdateGroupModelPriceByJSONString(`{"enterprise":{"video":1e309}}`))
}
