package controller

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const playgroundImageResponseCaptureLimit = 2 << 20

const (
	playgroundHistoryPromptMaxRunes        = 4096
	playgroundHistoryRevisedPromptMaxRunes = 1024
	playgroundHistoryImageURLMaxLength     = 4096
	playgroundHistoryMaxImagesPerRequest   = 4
)

type playgroundResponseWriter struct {
	gin.ResponseWriter
	body      bytes.Buffer
	maxSize   int
	truncated bool
}

func (writer *playgroundResponseWriter) Write(data []byte) (int, error) {
	if !writer.truncated {
		remaining := writer.maxSize - writer.body.Len()
		if len(data) <= remaining {
			_, _ = writer.body.Write(data)
		} else {
			writer.truncated = true
			writer.body.Reset()
		}
	}
	return writer.ResponseWriter.Write(data)
}

func (writer *playgroundResponseWriter) WriteString(data string) (int, error) {
	return writer.Write([]byte(data))
}

func preparePlayground(c *gin.Context, relayFormat types.RelayFormat) *types.NewAPIError {
	if c.GetBool("use_access_token") {
		return types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, nil, nil)
	if err != nil {
		return types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	userId := c.GetInt("id")
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
	}
	userCache.WriteContext(c)

	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("playground-%s", relayInfo.UsingGroup),
		Group:  relayInfo.UsingGroup,
	}
	if err = middleware.SetupContextForToken(c, tempToken); err != nil {
		return types.NewError(err, types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
	}
	return nil
}

func respondPlaygroundError(c *gin.Context, newAPIError *types.NewAPIError) {
	c.JSON(newAPIError.StatusCode, gin.H{
		"error": newAPIError.ToOpenAIError(),
	})
}

func Playground(c *gin.Context) {
	if newAPIError := preparePlayground(c, types.RelayFormatOpenAI); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	Relay(c, types.RelayFormatOpenAI)
}

func PlaygroundImage(c *gin.Context) {
	if newAPIError := preparePlayground(c, types.RelayFormatOpenAIImage); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	request := struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}{}
	_ = common.UnmarshalBodyReusable(c, &request)

	originalWriter := c.Writer
	capture := &playgroundResponseWriter{
		ResponseWriter: originalWriter,
		maxSize:        playgroundImageResponseCaptureLimit,
	}
	c.Writer = capture
	Relay(c, types.RelayFormatOpenAIImage)
	c.Writer = originalWriter

	if capture.truncated || capture.Status() < http.StatusOK || capture.Status() >= http.StatusMultipleChoices {
		return
	}
	if err := persistPlaygroundImageHistory(model.DB, c.GetInt("id"), request.Model, request.Prompt, capture.body.Bytes(), time.Now()); err != nil {
		common.SysError("insert playground image history error: " + err.Error())
	}
}

func persistPlaygroundImageHistory(db *gorm.DB, userID int, modelName string, prompt string, responseBody []byte, createdAt time.Time) error {
	history, err := buildPlaygroundImageHistory(userID, modelName, prompt, responseBody, createdAt)
	if err != nil {
		return err
	}
	return history.Insert(db)
}

func buildPlaygroundImageHistory(userID int, modelName string, prompt string, responseBody []byte, createdAt time.Time) (*model.PlaygroundMediaHistory, error) {
	response := dto.ImageResponse{}
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}
	images := make([]dto.ImageData, 0, playgroundHistoryMaxImagesPerRequest)
	for _, image := range response.Data {
		if len(images) >= playgroundHistoryMaxImagesPerRequest {
			break
		}
		if len(image.Url) == 0 || len(image.Url) > playgroundHistoryImageURLMaxLength {
			continue
		}
		parsedURL, err := url.Parse(image.Url)
		if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			continue
		}
		image.B64Json = ""
		image.RevisedPrompt = truncatePlaygroundHistoryText(image.RevisedPrompt, playgroundHistoryRevisedPromptMaxRunes)
		images = append(images, image)
	}
	if len(images) == 0 {
		return nil, errors.New("image response contains no results")
	}
	result, err := common.Marshal(images)
	if err != nil {
		return nil, err
	}
	return model.NewPlaygroundImageHistory(
		userID,
		truncatePlaygroundHistoryText(modelName, 191),
		truncatePlaygroundHistoryText(prompt, playgroundHistoryPromptMaxRunes),
		string(result),
		createdAt,
	), nil
}

func truncatePlaygroundHistoryText(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func PlaygroundVideo(c *gin.Context) {
	if newAPIError := preparePlayground(c, types.RelayFormatTask); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	request := relaycommon.TaskSubmitReq{}
	_ = common.UnmarshalBodyReusable(c, &request)
	c.Set("playground_prompt", truncatePlaygroundHistoryText(request.Prompt, playgroundHistoryPromptMaxRunes))
	RelayTask(c)
}

func PlaygroundVideoFetch(c *gin.Context) {
	if newAPIError := preparePlayground(c, types.RelayFormatTask); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	RelayTaskFetch(c)
}

func GetPlaygroundMediaHistory(c *gin.Context) {
	mediaType, valid := model.ParseMediaType(c.Query("media_type"))
	if !valid || mediaType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid media_type"})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 || limit > model.PlaygroundMediaHistoryMaxItems {
		limit = model.PlaygroundMediaHistoryMaxItems
	}
	userID := c.GetInt("id")
	now := time.Now().Unix()
	items := make([]*dto.PlaygroundMediaHistory, 0)

	if mediaType == model.MediaTypeImage {
		histories, err := model.GetPlaygroundImageHistory(model.DB, userID, now, limit)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		for _, history := range histories {
			images := make([]dto.ImageData, 0)
			if err = history.DecodeResult(&images); err != nil {
				common.SysError("decode playground image history error: " + err.Error())
				continue
			}
			items = append(items, &dto.PlaygroundMediaHistory{
				ID:          fmt.Sprintf("image_%d", history.ID),
				MediaType:   string(model.MediaTypeImage),
				Model:       history.Model,
				Prompt:      history.Prompt,
				Status:      "completed",
				Progress:    100,
				Images:      images,
				CreatedAt:   history.CreatedAt,
				CompletedAt: history.CreatedAt,
			})
		}
		common.ApiSuccess(c, items)
		return
	}

	tasks, err := model.GetPlaygroundVideoHistory(model.DB, userID, now, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, task := range tasks {
		items = append(items, playgroundVideoHistoryDTO(task))
	}
	common.ApiSuccess(c, items)
}

func playgroundVideoHistoryDTO(task *model.Task) *dto.PlaygroundMediaHistory {
	createdAt := task.CreatedAt
	if createdAt == 0 {
		createdAt = task.SubmitTime
	}
	completedAt := task.FinishTime
	if completedAt == 0 && (task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure) {
		completedAt = task.UpdatedAt
	}
	progress, _ := strconv.Atoi(strings.TrimSuffix(task.Progress, "%"))
	if task.Status == model.TaskStatusSuccess {
		progress = 100
	}

	item := &dto.PlaygroundMediaHistory{
		ID:          task.TaskID,
		MediaType:   string(model.MediaTypeVideo),
		Model:       task.Properties.OriginModelName,
		Prompt:      task.Properties.Input,
		TaskID:      task.TaskID,
		Status:      task.Status.ToVideoStatus(),
		Progress:    progress,
		CreatedAt:   createdAt,
		CompletedAt: completedAt,
	}
	if task.Status == model.TaskStatusSuccess && task.GetResultURL() != "" {
		item.ResultURL = "/v1/videos/" + task.TaskID + "/content"
	}
	if task.Status == model.TaskStatusFailure {
		item.Error = task.FailReason
	}
	return item
}
