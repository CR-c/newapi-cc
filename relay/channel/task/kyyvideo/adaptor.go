package kyyvideo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if taskErr := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if req.Duration == 0 && req.Seconds != "" {
		req.Duration, err = strconv.Atoi(req.Seconds)
		if err != nil {
			return service.TaskErrorWrapperLocal(fmt.Errorf("duration must be an integer"), "invalid_duration", http.StatusBadRequest)
		}
	}
	modelName := info.UpstreamModelName
	if modelName == "" {
		modelName = info.OriginModelName
	}
	if modelName == "" {
		modelName = req.Model
	}
	if err = validateKyyVideoRequest(&req, modelName); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	req.Seconds = ""
	c.Set("task_request", req)
	return nil
}

func validateKyyVideoRequest(req *relaycommon.TaskSubmitReq, modelName string) error {
	capabilities, ok := modelCapabilitiesByName[modelName]
	if !ok {
		return fmt.Errorf("unsupported model: %s", modelName)
	}
	if modelName != "videos" && utf8.RuneCountInString(req.Prompt) > 5000 {
		return fmt.Errorf("prompt must contain at most 5000 characters")
	}
	if req.Duration == 0 {
		return fmt.Errorf("duration is required")
	}
	if len(capabilities.exactDurations) > 0 {
		if !capabilities.exactDurations[req.Duration] {
			return fmt.Errorf("duration must be 10 or 15 seconds")
		}
	} else if req.Duration < capabilities.minDuration || req.Duration > capabilities.maxDuration {
		return fmt.Errorf("duration must be between %d and %d seconds", capabilities.minDuration, capabilities.maxDuration)
	}
	if req.AspectRatio == "" {
		req.AspectRatio = "16:9"
	}
	if !capabilities.aspectRatios[req.AspectRatio] {
		return fmt.Errorf("unsupported aspect_ratio for %s", modelName)
	}
	if req.Resolution == "" {
		req.Resolution = "720p"
	}
	if req.Resolution != "720p" {
		return fmt.Errorf("resolution must be 720p")
	}
	if len(req.Images) > capabilities.maxImages {
		return fmt.Errorf("%s supports at most %d reference images", modelName, capabilities.maxImages)
	}
	if len(req.Videos) > capabilities.maxVideos {
		if capabilities.maxVideos == 0 {
			return fmt.Errorf("%s does not support reference videos", modelName)
		}
		return fmt.Errorf("%s supports at most %d reference videos", modelName, capabilities.maxVideos)
	}
	if len(req.Audios) > capabilities.maxAudios {
		return fmt.Errorf("%s supports at most %d reference audios", modelName, capabilities.maxAudios)
	}
	if capabilities.requireImageAudio && len(req.Audios) > 0 && len(req.Images) == 0 {
		return fmt.Errorf("audio references require at least one reference image for %s", modelName)
	}

	hasFirstImage := strings.TrimSpace(req.FirstImage) != ""
	hasLastImage := strings.TrimSpace(req.LastImage) != ""
	if hasFirstImage != hasLastImage {
		return fmt.Errorf("first_image and last_image must be provided together")
	}
	if hasFirstImage && (len(req.Images) > 0 || len(req.Videos) > 0 || len(req.Audios) > 0) {
		return fmt.Errorf("first/last frame mode cannot be combined with reference media")
	}

	mediaURLs := append([]string{}, req.Images...)
	mediaURLs = append(mediaURLs, req.Videos...)
	mediaURLs = append(mediaURLs, req.Audios...)
	if hasFirstImage {
		mediaURLs = append(mediaURLs, req.FirstImage, req.LastImage)
	}
	for _, mediaURL := range mediaURLs {
		if err := taskcommon.ValidateMediaURL(mediaURL, false, taskcommon.MediaURLPortPolicyEnforceConfigured); err != nil {
			return fmt.Errorf("invalid media URL: %w", err)
		}
	}
	if modelName != "videos" {
		req.AutoFace = nil
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + CreateEndpoint, nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	modelName := info.UpstreamModelName
	if modelName == "" {
		modelName = req.Model
	}
	payload := requestPayload{
		Model:           modelName,
		Prompt:          req.Prompt,
		Duration:        req.Duration,
		Ratio:           req.AspectRatio,
		Resolution:      req.Resolution,
		FirstImage:      req.FirstImage,
		LastImage:       req.LastImage,
		ReferenceImages: append([]string(nil), req.Images...),
		ReferenceVideos: append([]string(nil), req.Videos...),
		ReferenceAudios: append([]string(nil), req.Audios...),
		AutoFace:        req.AutoFace,
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var upstream taskResponse
	if err = common.Unmarshal(responseBody, &upstream); err != nil {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("unmarshal KYY video response: %w", err), "invalid_response", http.StatusBadGateway)
	}
	if upstream.Status == "failed" || upstream.ID == "" {
		statusCode := resp.StatusCode
		if statusCode < http.StatusBadRequest {
			statusCode = http.StatusBadRequest
		}
		return "", a.SanitizeTaskData(responseBody), service.TaskErrorWrapper(
			fmt.Errorf("KYY video task creation failed"),
			"upstream_error",
			statusCode,
		)
	}

	clientResponse := dto.NewOpenAIVideo()
	clientResponse.ID = info.PublicTaskID
	clientResponse.TaskID = info.PublicTaskID
	clientResponse.Model = info.OriginModelName
	clientResponse.CreatedAt = upstream.Created
	clientResponse.Status = mapKyyVideoStatus(upstream.Status)
	c.JSON(http.StatusOK, clientResponse)
	return upstream.ID, a.SanitizeTaskData(responseBody), nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+QueryEndpoint+url.PathEscape(taskID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("KYY video query returned status %d", resp.StatusCode)
	}
	return resp, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var upstream taskResponse
	if err := common.Unmarshal(respBody, &upstream); err != nil {
		return nil, fmt.Errorf("unmarshal KYY video task result: %w", err)
	}
	if upstream.Status != "queued" && upstream.Status != "processing" && upstream.Status != "completed" && upstream.Status != "failed" {
		return nil, fmt.Errorf("KYY video task returned invalid status %q", upstream.Status)
	}
	if upstream.ID == "" && upstream.Status != "failed" {
		return nil, fmt.Errorf("KYY video task id is empty")
	}
	result := &relaycommon.TaskInfo{Code: 0, TaskID: upstream.ID}
	switch upstream.Status {
	case "queued":
		result.Status = model.TaskStatusQueued
		result.Progress = taskcommon.ProgressQueued
	case "processing":
		result.Status = model.TaskStatusInProgress
		result.Progress = taskcommon.ProgressInProgress
	case "completed":
		result.Status = model.TaskStatusSuccess
		result.Progress = taskcommon.ProgressComplete
		result.Url = upstream.VideoURL
	case "failed":
		result.Status = model.TaskStatusFailure
		result.Progress = taskcommon.ProgressComplete
		result.Reason = "KYY video generation failed"
	default:
		result.Status = model.TaskStatusInProgress
		result.Progress = taskcommon.ProgressSubmitted
	}
	return result, nil
}

func mapKyyVideoStatus(status string) string {
	switch status {
	case "processing":
		return dto.VideoStatusInProgress
	case "completed":
		return dto.VideoStatusCompleted
	case "failed":
		return dto.VideoStatusFailed
	default:
		return dto.VideoStatusQueued
	}
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	response := task.ToOpenAIVideo()
	if task.Status == model.TaskStatusSuccess {
		response.SetMetadata("url", taskcommon.BuildProxyURL(task.TaskID))
	}
	return common.Marshal(response)
}

func (a *TaskAdaptor) SanitizeTaskData(body []byte) []byte {
	var upstream taskResponse
	if err := common.Unmarshal(body, &upstream); err != nil {
		return nil
	}
	safe := struct {
		Object  string `json:"object,omitempty"`
		Created int64  `json:"created,omitempty"`
		Model   string `json:"model,omitempty"`
		Status  string `json:"status,omitempty"`
	}{
		Object:  upstream.Object,
		Created: upstream.Created,
		Model:   upstream.Model,
		Status:  upstream.Status,
	}
	data, err := common.Marshal(safe)
	if err != nil {
		return nil
	}
	return data
}

func (a *TaskAdaptor) GetModelList() []string {
	return append([]string(nil), ModelList...)
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}
