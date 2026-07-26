package constant

// Video asset groups share the asset:// media library and reference limits.
// Corn* groups are exclusive higher-ratio mirrors of the dddd product path.
const (
	VideoAssetGroupDefault     = "video-dddd"
	VideoAssetGroupSD          = "sd-dddd"
	VideoAssetGroupCorn        = "Corn专用"
	VideoAssetGroupCornVP00001 = "Corn-vp00001"
)

// IsVideoAssetGroup reports whether usingGroup may use asset:// references
// and the video asset library endpoints.
func IsVideoAssetGroup(group string) bool {
	switch group {
	case VideoAssetGroupDefault, VideoAssetGroupSD, VideoAssetGroupCorn, VideoAssetGroupCornVP00001:
		return true
	default:
		return false
	}
}
