package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
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

func TestNormalizeNewRedemptionFundingSource(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "missing defaults to paid", source: "", want: model.RedemptionFundingSourcePaid},
		{name: "paid card", source: model.RedemptionFundingSourcePaid, want: model.RedemptionFundingSourcePaid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeNewRedemptionFundingSource(tt.source)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	for _, source := range []string{model.RedemptionFundingSourcePromo, model.RedemptionFundingSourceLegacyUnknown, "cash", "PAID"} {
		_, err := normalizeNewRedemptionFundingSource(source)
		require.Error(t, err)
	}
}

func TestRedemptionMutationsRequireRoot(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	handlers := []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{name: "create", handler: AddRedemption},
		{name: "update", handler: UpdateRedemption},
		{name: "delete", handler: DeleteRedemption},
		{name: "delete invalid", handler: DeleteInvalidRedemption},
	}

	for _, tt := range handlers {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/redemption", nil)
			ctx.Set("role", common.RoleAdminUser)

			tt.handler(ctx)

			var response struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.False(t, response.Success)
			assert.NotEmpty(t, response.Message)
		})
	}
}
