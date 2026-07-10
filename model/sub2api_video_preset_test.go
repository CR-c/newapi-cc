package model

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestInitSub2APIVideoPresetFromEnvRequiresBaseURLAndKey(t *testing.T) {
	truncateTables(t)
	t.Setenv("SUB2API_VIDEO_PRESET_ENABLED", "true")
	t.Setenv("SUB2API_BASE_URL", "")
	t.Setenv("SUB2API_API_KEY", "sk-upstream")

	err := InitSub2APIVideoPresetFromEnv()
	require.ErrorContains(t, err, "SUB2API_BASE_URL")

	t.Setenv("SUB2API_BASE_URL", "https://sub2api.example")
	t.Setenv("SUB2API_API_KEY", "")
	err = InitSub2APIVideoPresetFromEnv()
	require.ErrorContains(t, err, "SUB2API_API_KEY")
}

func TestInitSub2APIVideoPresetFromEnvCreatesVideoGroupsAndChannels(t *testing.T) {
	truncateTables(t)
	t.Setenv("SUB2API_VIDEO_PRESET_ENABLED", "true")
	t.Setenv("SUB2API_BASE_URL", "https://sub2api.example/")
	t.Setenv("SUB2API_API_KEY", "sk-upstream")

	require.NoError(t, InitSub2APIVideoPresetFromEnv())

	var channels []Channel
	require.NoError(t, DB.Where("tag = ?", sub2APIVideoPresetTag).Order("name asc").Find(&channels).Error)
	require.Len(t, channels, len(sub2APIVideoPresetGroups))

	for _, channel := range channels {
		require.Equal(t, constant.ChannelTypeSora, channel.Type)
		require.Equal(t, "sk-upstream", channel.Key)
		require.Equal(t, "https://sub2api.example", channel.GetBaseURL())
		require.NotEmpty(t, channel.Group)
		require.NotEmpty(t, channel.Models)
	}

	for _, preset := range sub2APIVideoPresetGroups {
		var count int64
		require.NoError(t, DB.Model(&Ability{}).Where("`group` = ?", preset.Name).Count(&count).Error)
		require.Equal(t, int64(len(preset.Models)), count)
	}

	assertOptionJSONHasKey(t, "GroupRatio", "sub2api-jimeng-video")
	assertOptionJSONHasKey(t, "UserUsableGroups", "sub2api-jimeng-video")

	var prefill PrefillGroup
	require.NoError(t, DB.Where("name = ?", "sub2api-video-models").First(&prefill).Error)
	var models []string
	require.NoError(t, json.Unmarshal(prefill.Items, &models))
	require.Contains(t, models, "video-ds-2.0")
	require.Contains(t, models, "dreamina-seedance-2-0-fast-hc")
}

func TestInitSub2APIVideoPresetFromEnvIsIdempotent(t *testing.T) {
	truncateTables(t)
	t.Setenv("SUB2API_VIDEO_PRESET_ENABLED", "true")
	t.Setenv("SUB2API_BASE_URL", "https://sub2api.example")
	t.Setenv("SUB2API_API_KEY", "sk-first")

	require.NoError(t, InitSub2APIVideoPresetFromEnv())
	t.Setenv("SUB2API_API_KEY", "sk-second")
	require.NoError(t, InitSub2APIVideoPresetFromEnv())

	var channels []Channel
	require.NoError(t, DB.Where("tag = ?", sub2APIVideoPresetTag).Find(&channels).Error)
	require.Len(t, channels, len(sub2APIVideoPresetGroups))
	for _, channel := range channels {
		require.Equal(t, "sk-second", channel.Key)
	}

	var prefillCount int64
	require.NoError(t, DB.Model(&PrefillGroup{}).Where("name = ?", "sub2api-video-models").Count(&prefillCount).Error)
	require.Equal(t, int64(1), prefillCount)
}

func TestInitSub2APIVideoPresetFromEnvDisabledDoesNothing(t *testing.T) {
	truncateTables(t)
	require.NoError(t, os.Unsetenv("SUB2API_VIDEO_PRESET_ENABLED"))
	t.Setenv("SUB2API_BASE_URL", "https://sub2api.example")
	t.Setenv("SUB2API_API_KEY", "sk-upstream")

	require.NoError(t, InitSub2APIVideoPresetFromEnv())

	var channelCount int64
	require.NoError(t, DB.Model(&Channel{}).Where("tag = ?", sub2APIVideoPresetTag).Count(&channelCount).Error)
	require.Zero(t, channelCount)
}

func assertOptionJSONHasKey(t *testing.T, optionKey string, expectedKey string) {
	t.Helper()
	var option Option
	require.NoError(t, DB.Where("`key` = ?", optionKey).First(&option).Error)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(option.Value), &parsed))
	require.Contains(t, parsed, expectedKey)
}
