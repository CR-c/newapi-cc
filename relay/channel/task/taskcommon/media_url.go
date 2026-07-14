package taskcommon

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const maxTaskMediaDataURLLength = 15 << 20

func ValidateMediaURL(value string, allowImageDataURL bool) error {
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

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("media URL must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil {
		return fmt.Errorf("media URL must not contain credentials")
	}
	fetchSetting := system_setting.GetFetchSetting()
	return common.ValidateURLWithFetchSetting(
		value,
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	)
}
