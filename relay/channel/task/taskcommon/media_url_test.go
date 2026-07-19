package taskcommon

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
)

func TestValidateMediaURLRejectsInternalAssetAndUnsupportedDataURLs(t *testing.T) {
	assert.Error(t, ValidateMediaURL("asset://shared-upstream-asset", false, MediaURLPortPolicyEnforceConfigured))
	assert.Error(t, ValidateMediaURL("file:///etc/passwd", false, MediaURLPortPolicyEnforceConfigured))
	assert.Error(t, ValidateMediaURL("data:image/png;base64,aGVsbG8=", false, MediaURLPortPolicyEnforceConfigured))
	assert.Error(t, ValidateMediaURL("data:text/html;base64,aGVsbG8=", true, MediaURLPortPolicyEnforceConfigured))
	assert.NoError(t, ValidateMediaURL("data:image/png;base64,aGVsbG8=", true, MediaURLPortPolicyEnforceConfigured))
	assert.ErrorContains(t, ValidateMediaURL("https://example.com/"+strings.Repeat("a", 4096), false, MediaURLPortPolicyEnforceConfigured), "too long")
}

func TestValidateMediaURLCustomPortPolicyPreservesPrivateIPProtection(t *testing.T) {
	assert.ErrorContains(t, ValidateMediaURL("https://1.1.1.1:90/ref.png", false, MediaURLPortPolicyEnforceConfigured), "port 90 is not allowed")
	assert.ErrorContains(t, ValidateMediaURL("https://1.1.1.1:90/ref.png", false, MediaURLPortPolicyAllowCustomDomain), "port 90 is not allowed")
	assert.ErrorContains(t, ValidateMediaURL("https://127.0.0.1:443/ref.png", false, MediaURLPortPolicyAllowCustomDomain), "private IP address not allowed")
	assert.ErrorContains(t, ValidateMediaURL("https://[::1]:443/ref.png", false, MediaURLPortPolicyAllowCustomDomain), "private IP address not allowed")
}

func TestValidateMediaURLCustomDomainPortRetainsDomainAndCredentialFilters(t *testing.T) {
	fetchSetting := system_setting.GetFetchSetting()
	originalFetchSetting := *fetchSetting
	t.Cleanup(func() {
		*fetchSetting = originalFetchSetting
	})
	fetchSetting.EnableSSRFProtection = true
	fetchSetting.AllowPrivateIp = false
	fetchSetting.DomainFilterMode = false
	fetchSetting.IpFilterMode = false
	fetchSetting.DomainList = []string{"blocked.example"}
	fetchSetting.IpList = nil
	fetchSetting.AllowedPorts = []string{"443"}
	fetchSetting.ApplyIPFilterForDomain = false

	assert.NoError(t, ValidateMediaURL("https://allowed.example:90/ref.png", false, MediaURLPortPolicyAllowCustomDomain))
	assert.ErrorContains(t, ValidateMediaURL("http://allowed.example:90/ref.png", false, MediaURLPortPolicyAllowCustomDomain), "port 90 is not allowed")
	assert.ErrorContains(t, ValidateMediaURL("https://blocked.example:90/ref.png", false, MediaURLPortPolicyAllowCustomDomain), "domain in blacklist")
	assert.ErrorContains(t, ValidateMediaURL("https://user:pass@allowed.example:90/ref.png", false, MediaURLPortPolicyAllowCustomDomain), "must not contain credentials")
	for _, ambiguousIP := range []string{"2130706433", "127.1", "0x7f000001", "16843009"} {
		assert.ErrorContains(t, ValidateMediaURL("https://"+ambiguousIP+":90/ref.png", false, MediaURLPortPolicyAllowCustomDomain), "port 90 is not allowed")
	}
	assert.NoError(t, ValidateMediaURL("https://allowed.example.:443/ref.png", false, MediaURLPortPolicyEnforceConfigured))
	assert.NoError(t, ValidateMediaURL("https://intranet:443/ref.png", false, MediaURLPortPolicyEnforceConfigured))
	fetchSetting.AllowedPorts = []string{"8443"}
	assert.ErrorContains(t, ValidateMediaURL("https://allowed.example/ref.png", false, MediaURLPortPolicyAllowCustomDomain), "port 443 is not allowed")
}
