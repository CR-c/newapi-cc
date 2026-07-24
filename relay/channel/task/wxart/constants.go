package wxart

const (
	ChannelName    = "wxart"
	CreateEndpoint = "/v1/videos"
	QueryEndpoint  = "/v1/videos/"
)

var ModelList = []string{
	ModelImagine15,
	ModelVideo3,
}

const (
	ModelImagine15 = "grok-imagine-video-1.5-preview"
	ModelVideo3    = "grok-video-3"
)

type modelCapabilities struct {
	// min/max duration when exactDurations is empty
	minDuration    int
	maxDuration    int
	exactDurations map[int]bool
	aspectRatios   map[string]bool
	// modeRequired: upstream requires mode=text|frame|ref
	modeRequired bool
	// maxImages for frame/ref; 0 images allowed only in text mode
	maxImages int
	// requireExactlyOneImage when true (imagine 1.5) — always needs 1 image, no mode
	requireExactlyOneImage bool
	// useSizeField: 1.5 accepts size; video-3 requires resolution and rejects size
	useSizeField    bool
	defaultDuration int
	defaultRatio    string
	defaultRes      string
}

var modelCapabilitiesByName = map[string]modelCapabilities{
	ModelImagine15: {
		minDuration:            1,
		maxDuration:            15,
		aspectRatios:           map[string]bool{"16:9": true, "9:16": true, "1:1": true, "4:3": true, "3:4": true, "3:2": true, "2:3": true},
		maxImages:              1,
		requireExactlyOneImage: true,
		useSizeField:           true,
		defaultDuration:        6,
		defaultRatio:           "16:9",
		defaultRes:             "720p",
	},
	ModelVideo3: {
		exactDurations:  map[int]bool{6: true, 10: true, 12: true, 16: true, 20: true},
		aspectRatios:    map[string]bool{"16:9": true, "9:16": true, "1:1": true},
		modeRequired:    true,
		maxImages:       7,
		useSizeField:    false,
		defaultDuration: 6,
		defaultRatio:    "16:9",
		defaultRes:      "720p",
	},
}
