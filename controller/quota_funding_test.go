package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAdminQuotaCreditBucketDefaultsToPromo(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   model.WalletBucket
	}{
		{name: "missing", source: "", want: model.WalletBucketPromo},
		{name: "promo", source: "promo", want: model.WalletBucketPromo},
		{name: "paid", source: "paid", want: model.WalletBucketPaid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bucket, err := resolveAdminQuotaCreditBucket(test.source)
			require.NoError(t, err)
			assert.Equal(t, test.want, bucket)
		})
	}

	_, err := resolveAdminQuotaCreditBucket("legacy_unknown")
	require.Error(t, err)
}

func TestValidateRedemptionQuotaBounds(t *testing.T) {
	require.Error(t, validateRedemptionQuota(0))
	require.Error(t, validateRedemptionQuota(-1))
	require.NoError(t, validateRedemptionQuota(1))
	require.NoError(t, validateRedemptionQuota(common.MaxQuota))
	require.Error(t, validateRedemptionQuota(common.MaxQuota+1))
}
