package common

import "strings"

const (
	VideoCostTier480p720pNoReference = "video:480p-720p:no-reference"
	VideoCostTier480p720pReference   = "video:480p-720p:reference"
	VideoCostTier1080pNoReference    = "video:1080p:no-reference"
	VideoCostTier1080pReference      = "video:1080p:reference"
	VideoCostTier4KNoReference       = "video:4k:no-reference"
	VideoCostTier4KReference         = "video:4k:reference"
)

func VideoCostTier(resolution string, hasReference bool) string {
	resolutionTier := "480p-720p"
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "1080p":
		resolutionTier = "1080p"
	case "4k":
		resolutionTier = "4k"
	}
	referenceTier := "no-reference"
	if hasReference {
		referenceTier = "reference"
	}
	return "video:" + resolutionTier + ":" + referenceTier
}
