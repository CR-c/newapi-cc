package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAmountBonusRequiresExactPositivePreset(t *testing.T) {
	original := paymentSetting.AmountBonus
	paymentSetting.AmountBonus = map[int]int{
		100: 10,
		200: 0,
		500: -5,
	}
	t.Cleanup(func() { paymentSetting.AmountBonus = original })

	assert.Equal(t, 10, GetAmountBonus(100))
	assert.Zero(t, GetAmountBonus(99))
	assert.Zero(t, GetAmountBonus(200))
	assert.Zero(t, GetAmountBonus(500))
	assert.Zero(t, GetAmountBonus(-1))
}
