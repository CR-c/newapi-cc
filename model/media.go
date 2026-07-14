package model

import (
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

type MediaType string

type ModelCapability string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"

	ModelCapabilityChat  ModelCapability = "chat"
	ModelCapabilityImage ModelCapability = "image"
	ModelCapabilityVideo ModelCapability = "video"
)

func ParseMediaType(value string) (MediaType, bool) {
	switch MediaType(value) {
	case "":
		return "", true
	case MediaTypeImage, MediaTypeVideo:
		return MediaType(value), true
	default:
		return "", false
	}
}

func ParseModelCapability(value string) (ModelCapability, bool) {
	switch ModelCapability(value) {
	case "":
		return "", true
	case ModelCapabilityChat, ModelCapabilityImage, ModelCapabilityVideo:
		return ModelCapability(value), true
	default:
		return "", false
	}
}

func GetGroupEnabledCapabilityModels(group string, capability ModelCapability) []string {
	models := GetGroupEnabledModels(group)
	if capability == "" {
		return models
	}

	GetPricing()
	filtered := make([]string, 0, len(models))
	for _, modelName := range models {
		if modelSupportsCapability(modelName, capability) {
			filtered = append(filtered, modelName)
		}
	}
	return filtered
}

func modelSupportsCapability(modelName string, capability ModelCapability) bool {
	for _, endpointType := range GetModelSupportEndpointTypes(modelName) {
		switch capability {
		case ModelCapabilityImage:
			if endpointType == constant.EndpointTypeImageGeneration {
				return true
			}
		case ModelCapabilityVideo:
			if endpointType == constant.EndpointTypeOpenAIVideo {
				return true
			}
		case ModelCapabilityChat:
			if endpointType == constant.EndpointTypeOpenAI {
				return true
			}
		}
	}
	return false
}

func ApplyMediaTypeLogFilter(tx *gorm.DB, mediaType MediaType) *gorm.DB {
	if mediaType == "" {
		return tx
	}
	return tx.Where("logs.other LIKE ?", `%"media_type"%"`+string(mediaType)+`"%`)
}
