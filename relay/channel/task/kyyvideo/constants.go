package kyyvideo

const (
	ChannelName    = "kyy-video"
	CreateEndpoint = "/v1/videos/videos"
	QueryEndpoint  = "/v1/result/"
)

var ModelList = []string{
	"videos",
	"videos_stable",
	"videos_stable_fast",
	"videos_pro",
	"videos_pro_fast",
}

type modelCapabilities struct {
	minDuration       int
	maxDuration       int
	exactDurations    map[int]bool
	aspectRatios      map[string]bool
	maxImages         int
	maxVideos         int
	maxAudios         int
	requireImageAudio bool
}

var modelCapabilitiesByName = map[string]modelCapabilities{
	"videos": {
		minDuration:  4,
		maxDuration:  15,
		aspectRatios: map[string]bool{"21:9": true, "16:9": true, "4:3": true, "1:1": true, "3:4": true, "9:16": true},
		maxImages:    9,
		maxVideos:    3,
		maxAudios:    3,
	},
	"videos_stable": {
		minDuration:  4,
		maxDuration:  15,
		aspectRatios: map[string]bool{"16:9": true, "9:16": true, "1:1": true},
		maxImages:    4,
		maxVideos:    3,
		maxAudios:    1,
	},
	"videos_stable_fast": {
		exactDurations: map[int]bool{10: true, 15: true},
		aspectRatios:   map[string]bool{"16:9": true, "9:16": true, "1:1": true},
		maxImages:      4,
		maxVideos:      3,
		maxAudios:      1,
	},
	"videos_pro": {
		exactDurations:    map[int]bool{10: true, 15: true},
		aspectRatios:      map[string]bool{"16:9": true, "9:16": true, "1:1": true},
		maxImages:         9,
		maxAudios:         3,
		requireImageAudio: true,
	},
	"videos_pro_fast": {
		exactDurations:    map[int]bool{10: true, 15: true},
		aspectRatios:      map[string]bool{"16:9": true, "9:16": true, "1:1": true},
		maxImages:         9,
		maxAudios:         3,
		requireImageAudio: true,
	},
}
