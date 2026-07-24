package constant

// Video asset groups share the asset:// media library and reference limits.
// Corn专用 is an exclusive higher-ratio mirror of sd-dddd (same product path).
const (
	VideoAssetGroupDefault = "video-dddd"
	VideoAssetGroupSD      = "sd-dddd"
	VideoAssetGroupCorn    = "Corn专用"
)

// IsVideoAssetGroup reports whether usingGroup may use asset:// references
// and the video asset library endpoints.
func IsVideoAssetGroup(group string) bool {
	switch group {
	case VideoAssetGroupDefault, VideoAssetGroupSD, VideoAssetGroupCorn:
		return true
	default:
		return false
	}
}
