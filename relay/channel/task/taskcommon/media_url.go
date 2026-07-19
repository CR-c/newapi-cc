package taskcommon

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const (
	maxTaskMediaURLLength     = 4096
	maxTaskMediaDataURLLength = 15 << 20
)

// MediaURLPortPolicy controls whether a forwarded media URL may use a custom domain port.
type MediaURLPortPolicy int

const (
	MediaURLPortPolicyEnforceConfigured MediaURLPortPolicy = iota
	MediaURLPortPolicyAllowCustomDomain
)

func isStrictDNSHostname(host string) bool {
	if len(host) == 0 || len(host) > 253 || strings.HasSuffix(host, ".") {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			char := label[i]
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	for _, char := range labels[len(labels)-1] {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			return true
		}
	}
	return false
}

func ValidateMediaURL(value string, allowImageDataURL bool, portPolicy MediaURLPortPolicy) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("media URL is required")
	}
	if strings.HasPrefix(value, "data:") {
		if !allowImageDataURL {
			return fmt.Errorf("data URLs are not supported for this media field")
		}
		if len(value) > maxTaskMediaDataURLLength {
			return fmt.Errorf("image data URL is too large")
		}
		parts := strings.SplitN(value, ",", 2)
		if len(parts) != 2 || !strings.HasSuffix(parts[0], ";base64") {
			return fmt.Errorf("image data URL must use base64 encoding")
		}
		mediaType := strings.TrimSuffix(strings.TrimPrefix(parts[0], "data:"), ";base64")
		switch strings.ToLower(mediaType) {
		case "image/jpeg", "image/png", "image/webp":
		default:
			return fmt.Errorf("unsupported image data URL content type")
		}
		if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
			return fmt.Errorf("invalid image data URL: %w", err)
		}
		return nil
	}
	if len(value) > maxTaskMediaURLLength {
		return fmt.Errorf("media URL is too long")
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("media URL must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil {
		return fmt.Errorf("media URL must not contain credentials")
	}
	host := parsed.Hostname()
	literalIP := net.ParseIP(host)
	fetchSetting := system_setting.GetFetchSetting()
	allowedPorts := fetchSetting.AllowedPorts
	if portPolicy == MediaURLPortPolicyAllowCustomDomain && parsed.Scheme == "https" && parsed.Port() != "" && literalIP == nil && isStrictDNSHostname(host) {
		allowedPorts = nil
	}
	return common.ValidateURLWithFetchSetting(
		value,
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		allowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	)
}
