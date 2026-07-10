package model

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const sub2APIVideoPresetTag = "sub2api-video-preset"

type sub2APIVideoPresetGroup struct {
	Name        string
	Description string
	Models      []string
}

var sub2APIVideoPresetGroups = []sub2APIVideoPresetGroup{
	{
		Name:        "sub2api-jimeng-video",
		Description: "Sub2API 即梦视频",
		Models:      []string{"video-ds-2.0", "video-ds-2.0-fast", "as-sd2.0-fast"},
	},
	{
		Name:        "sub2api-grok-video",
		Description: "Sub2API Grok 视频",
		Models:      []string{"grok-imagine-video", "grok-imagine-video-1.5-preview"},
	},
	{
		Name:        "sub2api-grok-video-per-request",
		Description: "Sub2API Grok 视频按次",
		Models:      []string{"grok-image-video", "grok-video-1.5"},
	},
	{
		Name:        "sub2api-jimeng-nsfw-video",
		Description: "Sub2API 即梦 NSFW 视频",
		Models:      []string{"dreamina-seedance-2-0-hc", "dreamina-seedance-2-0-fast-hc"},
	},
}

func InitSub2APIVideoPresetFromEnv() error {
	if !isEnvEnabled("SUB2API_VIDEO_PRESET_ENABLED") {
		return nil
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SUB2API_BASE_URL")), "/")
	apiKey := strings.TrimSpace(os.Getenv("SUB2API_API_KEY"))
	if baseURL == "" {
		return fmt.Errorf("SUB2API_BASE_URL is required when SUB2API_VIDEO_PRESET_ENABLED=true")
	}
	if apiKey == "" {
		return fmt.Errorf("SUB2API_API_KEY is required when SUB2API_VIDEO_PRESET_ENABLED=true")
	}

	if err := upsertSub2APIVideoOptions(); err != nil {
		return err
	}
	if err := upsertSub2APIVideoPrefillGroup(); err != nil {
		return err
	}
	if err := upsertSub2APIVideoChannels(baseURL, apiKey); err != nil {
		return err
	}

	if common.MemoryCacheEnabled {
		InitChannelCache()
	}
	common.SysLog("sub2api video preset initialized")
	return nil
}

func isEnvEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func upsertSub2APIVideoOptions() error {
	groupRatio := ratio_setting.GetGroupRatioCopy()
	userUsableGroups := setting.GetUserUsableGroupsCopy()
	autoGroupSet := make(map[string]struct{})
	for _, group := range setting.GetAutoGroups() {
		if strings.TrimSpace(group) != "" {
			autoGroupSet[group] = struct{}{}
		}
	}

	for _, preset := range sub2APIVideoPresetGroups {
		groupRatio[preset.Name] = 1
		userUsableGroups[preset.Name] = preset.Description
		autoGroupSet[preset.Name] = struct{}{}
	}

	autoGroups := make([]string, 0, len(autoGroupSet))
	for group := range autoGroupSet {
		autoGroups = append(autoGroups, group)
	}
	sort.Strings(autoGroups)

	groupRatioJSON, err := json.Marshal(groupRatio)
	if err != nil {
		return err
	}
	userUsableGroupsJSON, err := json.Marshal(userUsableGroups)
	if err != nil {
		return err
	}
	autoGroupsJSON, err := json.Marshal(autoGroups)
	if err != nil {
		return err
	}

	return UpdateOptionsBulk(map[string]string{
		"GroupRatio":       string(groupRatioJSON),
		"UserUsableGroups": string(userUsableGroupsJSON),
		"AutoGroups":       string(autoGroupsJSON),
	})
}

func upsertSub2APIVideoPrefillGroup() error {
	models := make([]string, 0)
	seen := make(map[string]struct{})
	for _, preset := range sub2APIVideoPresetGroups {
		for _, modelName := range preset.Models {
			if _, ok := seen[modelName]; ok {
				continue
			}
			seen[modelName] = struct{}{}
			models = append(models, modelName)
		}
	}
	sort.Strings(models)
	items, err := json.Marshal(models)
	if err != nil {
		return err
	}

	now := common.GetTimestamp()
	prefill := PrefillGroup{
		Name:        "sub2api-video-models",
		Type:        "model",
		Items:       JSONValue(items),
		Description: "Models exposed by the Sub2API video preset",
		CreatedTime: now,
		UpdatedTime: now,
	}

	var existing PrefillGroup
	if err := DB.Where("name = ?", prefill.Name).First(&existing).Error; err == nil {
		existing.Type = prefill.Type
		existing.Items = prefill.Items
		existing.Description = prefill.Description
		existing.UpdatedTime = now
		return DB.Save(&existing).Error
	}
	return DB.Create(&prefill).Error
}

func upsertSub2APIVideoChannels(baseURL string, apiKey string) error {
	tag := sub2APIVideoPresetTag
	base := baseURL
	priority := int64(100)
	weight := uint(10)
	autoBan := 0
	modelMapping := "{}"
	statusCodeMapping := "{}"

	for _, preset := range sub2APIVideoPresetGroups {
		channel := Channel{
			Type:              constant.ChannelTypeSora,
			Key:               apiKey,
			Status:            common.ChannelStatusEnabled,
			Name:              "Sub2API - " + preset.Description,
			Weight:            &weight,
			CreatedTime:       common.GetTimestamp(),
			BaseURL:           &base,
			Models:            strings.Join(preset.Models, ","),
			Group:             preset.Name,
			ModelMapping:      &modelMapping,
			StatusCodeMapping: &statusCodeMapping,
			Priority:          &priority,
			AutoBan:           &autoBan,
			Tag:               &tag,
		}

		var existing Channel
		if err := DB.Where("tag = ? AND name = ?", tag, channel.Name).First(&existing).Error; err == nil {
			existing.Type = channel.Type
			existing.Key = channel.Key
			existing.Status = channel.Status
			existing.Weight = channel.Weight
			existing.BaseURL = channel.BaseURL
			existing.Models = channel.Models
			existing.Group = channel.Group
			existing.ModelMapping = channel.ModelMapping
			existing.StatusCodeMapping = channel.StatusCodeMapping
			existing.Priority = channel.Priority
			existing.AutoBan = channel.AutoBan
			if err := existing.Update(); err != nil {
				return err
			}
			continue
		}

		if err := channel.Insert(); err != nil {
			return err
		}
	}
	return nil
}
