package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

type MediaType string

type ModelCapability string

const (
	PlaygroundMediaHistoryTTLSeconds = 24 * 60 * 60
	PlaygroundMediaHistoryMaxItems   = 50

	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"

	ModelCapabilityChat  ModelCapability = "chat"
	ModelCapabilityImage ModelCapability = "image"
	ModelCapabilityVideo ModelCapability = "video"
)

type PlaygroundMediaHistory struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int       `json:"user_id" gorm:"index"`
	MediaType MediaType `json:"media_type" gorm:"type:varchar(20);index"`
	Model     string    `json:"model" gorm:"type:varchar(191)"`
	Prompt    string    `json:"prompt" gorm:"type:text"`
	Result    string    `json:"-" gorm:"type:text"`
	CreatedAt int64     `json:"created_at" gorm:"index"`
	ExpiresAt int64     `json:"expires_at" gorm:"index"`
}

func (history *PlaygroundMediaHistory) DecodeResult(target any) error {
	return common.UnmarshalJsonStr(history.Result, target)
}

func (history *PlaygroundMediaHistory) Insert(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at <= ?", time.Now().Unix()).Delete(&PlaygroundMediaHistory{}).Error; err != nil {
			return err
		}
		if err := tx.Create(history).Error; err != nil {
			return err
		}

		var staleIDs []int64
		if err := tx.Model(&PlaygroundMediaHistory{}).
			Where("user_id = ? AND media_type = ?", history.UserID, history.MediaType).
			Order("created_at DESC, id DESC").
			Offset(PlaygroundMediaHistoryMaxItems).
			Limit(PlaygroundMediaHistoryMaxItems).
			Pluck("id", &staleIDs).Error; err != nil {
			return err
		}
		if len(staleIDs) == 0 {
			return nil
		}
		return tx.Delete(&PlaygroundMediaHistory{}, staleIDs).Error
	})
}

func GetPlaygroundImageHistory(db *gorm.DB, userID int, now int64, limit int) ([]*PlaygroundMediaHistory, error) {
	if limit <= 0 || limit > PlaygroundMediaHistoryMaxItems {
		limit = PlaygroundMediaHistoryMaxItems
	}
	if err := db.Where("expires_at <= ?", now).Delete(&PlaygroundMediaHistory{}).Error; err != nil {
		return nil, err
	}

	items := make([]*PlaygroundMediaHistory, 0)
	err := db.Where("user_id = ? AND media_type = ? AND created_at >= ? AND expires_at > ?",
		userID, MediaTypeImage, now-PlaygroundMediaHistoryTTLSeconds, now).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func GetPlaygroundVideoHistory(db *gorm.DB, userID int, now int64, limit int) ([]*Task, error) {
	if limit <= 0 || limit > PlaygroundMediaHistoryMaxItems {
		limit = PlaygroundMediaHistoryMaxItems
	}

	candidates := make([]*Task, 0)
	err := db.Select("id", "created_at", "updated_at", "task_id", "platform", "user_id", "status", "fail_reason", "submit_time", "finish_time", "progress", "properties", "private_data").
		Where("user_id = ? AND created_at > ?", userID, now-PlaygroundMediaHistoryTTLSeconds).
		Where("platform NOT IN ?", []constant.TaskPlatform{constant.TaskPlatformSuno, constant.TaskPlatformMidjourney}).
		Order("created_at DESC, id DESC").
		Limit(limit * 4).
		Find(&candidates).Error
	if err != nil {
		return nil, err
	}
	items := make([]*Task, 0, limit)
	for _, task := range candidates {
		// Input is the compatibility marker for playground tasks created before
		// is_playground was persisted explicitly.
		if !task.Properties.IsPlayground && task.Properties.Input == "" {
			continue
		}
		items = append(items, task)
		if len(items) == limit {
			break
		}
	}
	return items, err
}

func NewPlaygroundImageHistory(userID int, modelName string, prompt string, result string, createdAt time.Time) *PlaygroundMediaHistory {
	createdUnix := createdAt.Unix()
	return &PlaygroundMediaHistory{
		UserID:    userID,
		MediaType: MediaTypeImage,
		Model:     modelName,
		Prompt:    prompt,
		Result:    result,
		CreatedAt: createdUnix,
		ExpiresAt: createdUnix + PlaygroundMediaHistoryTTLSeconds,
	}
}

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
	compact := `%"media_type":"` + string(mediaType) + `"%`
	spaced := `%"media_type": "` + string(mediaType) + `"%`
	return tx.Where("(logs.other LIKE ? OR logs.other LIKE ?)", compact, spaced)
}
