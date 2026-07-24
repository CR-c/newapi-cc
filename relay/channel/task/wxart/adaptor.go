package wxart

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	displayModelName := info.OriginModelName
	if displayModelName == "" {
		displayModelName = req.Model
	}
	if displayModelName == "" {
		displayModelName = modelName
	}
	if err = validateWxartVideoRequest(&req, modelName, displayModelName); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	req.Seconds = ""
	c.Set("task_request", req)
	return nil
}

func validateWxartVideoRequest(req *relaycommon.TaskSubmitReq, modelName string, displayModelName string) error {
	capabilities, ok := modelCapabilitiesByName[modelName]
	if !ok {
		return fmt.Errorf("unsupported model: %s", displayModelName)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if len(req.Videos) > 0 || len(req.Audios) > 0 {
		return fmt.Errorf("%s only supports reference images", displayModelName)
	}

	// Collect images from images / image / first_image (single-frame alias)
	images := append([]string(nil), req.Images...)
	if strings.TrimSpace(req.Image) != "" {
		images = append(images, req.Image)
	}
	if strings.TrimSpace(req.FirstImage) != "" && len(images) == 0 {
		images = append(images, req.FirstImage)
	}
	req.Images = images
	req.Image = ""
	req.FirstImage = ""
	req.LastImage = ""

	if req.Duration == 0 {
		req.Duration = capabilities.defaultDuration
	}
	if len(capabilities.exactDurations) > 0 {
		if !capabilities.exactDurations[req.Duration] {
			return fmt.Errorf("duration must be one of 6, 10, 12, 16, 20")
		}
	} else if req.Duration < capabilities.minDuration || req.Duration > capabilities.maxDuration {
		return fmt.Errorf("duration must be between %d and %d seconds", capabilities.minDuration, capabilities.maxDuration)
	}

	if req.AspectRatio == "" {
		req.AspectRatio = capabilities.defaultRatio
	}
	if !capabilities.aspectRatios[req.AspectRatio] {
		return fmt.Errorf("unsupported aspect_ratio for %s", displayModelName)
	}

	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		resolution = strings.ToLower(strings.TrimSpace(req.Size))
	}
	if resolution == "" {
		resolution = capabilities.defaultRes
	}
	if resolution != "480p" && resolution != "720p" {
		return fmt.Errorf("resolution must be 480p or 720p")
	}
	req.Resolution = resolution
	req.Size = ""

	if capabilities.requireExactlyOneImage {
		if len(req.Images) != 1 {
			return fmt.Errorf("%s requires exactly one reference image", displayModelName)
		}
	} else if capabilities.modeRequired {
		mode := strings.ToLower(strings.TrimSpace(req.Mode))
		if mode == "" {
			// Infer mode from image count for unified client ergonomics.
			switch len(req.Images) {
			case 0:
				mode = "text"
			case 1:
				mode = "frame"
			default:
				mode = "ref"
			}
		}
		switch mode {
		case "text":
			if len(req.Images) != 0 {
				return fmt.Errorf("%s text mode does not accept images", displayModelName)
			}
		case "frame":
			if len(req.Images) != 1 {
				return fmt.Errorf("%s frame mode requires exactly 1 reference image", displayModelName)
			}
		case "ref":
			if len(req.Images) < 1 || len(req.Images) > capabilities.maxImages {
				return fmt.Errorf("%s ref mode requires 1 to %d reference images", displayModelName, capabilities.maxImages)
			}
		default:
			return fmt.Errorf("mode must be text, frame, or ref")
		}
		req.Mode = mode
	}

	for _, imageURL := range req.Images {
		if err := taskcommon.ValidateMediaURL(imageURL, false, taskcommon.MediaURLPortPolicyEnforceConfigured); err != nil {
			return fmt.Errorf("invalid reference image: %w", err)
		}
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
	capabilities := modelCapabilitiesByName[modelName]

	payload := requestPayload{
		Model:       modelName,
		Prompt:      req.Prompt,
		AspectRatio: req.AspectRatio,
		Duration:    req.Duration,
	}
	if len(req.Images) > 0 {
		payload.ImagesURL = append([]string(nil), req.Images...)
	}
	if capabilities.modeRequired && req.Mode != "" {
		payload.Mode = req.Mode
	}
	if capabilities.useSizeField {
		payload.Size = req.Resolution
	} else {
		payload.Resolution = req.Resolution
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
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("unmarshal wxart video response: %w", err), "invalid_response", http.StatusBadGateway)
	}
	if upstream.Error != nil && upstream.Error.Message != "" {
		statusCode := resp.StatusCode
		if statusCode < http.StatusBadRequest {
			statusCode = http.StatusBadRequest
		}
		return "", a.SanitizeTaskData(responseBody), service.TaskErrorWrapper(
			fmt.Errorf("%s", upstream.Error.Message),
			"upstream_error",
			statusCode,
		)
	}

	upstreamID := upstream.ID
	if upstreamID == "" {
		upstreamID = upstream.TaskID
	}
	if upstreamID == "" || strings.EqualFold(upstream.Status, "failed") {
		statusCode := resp.StatusCode
		if statusCode < http.StatusBadRequest {
			statusCode = http.StatusBadRequest
		}
		return "", a.SanitizeTaskData(responseBody), service.TaskErrorWrapper(
			fmt.Errorf("wxart video task creation failed"),
			"upstream_error",
			statusCode,
		)
	}

	clientResponse := dto.NewOpenAIVideo()
	clientResponse.ID = info.PublicTaskID
	clientResponse.TaskID = info.PublicTaskID
	clientResponse.Model = info.OriginModelName
	if upstream.CreatedAt > 0 {
		clientResponse.CreatedAt = upstream.CreatedAt
	}
	clientResponse.Status = mapWxartStatus(upstream.Status)
	c.JSON(http.StatusOK, clientResponse)
	return upstreamID, a.SanitizeTaskData(responseBody), nil
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
		return nil, fmt.Errorf("wxart video query returned status %d", resp.StatusCode)
	}
	return resp, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var upstream taskResponse
	if err := common.Unmarshal(respBody, &upstream); err != nil {
		return nil, fmt.Errorf("unmarshal wxart video task result: %w", err)
	}
	if upstream.Error != nil && upstream.Error.Message != "" &&
		!isActiveStatus(upstream.Status) && !strings.EqualFold(upstream.Status, "completed") {
		return &relaycommon.TaskInfo{
			Code:     0,
			TaskID:   firstNonEmpty(upstream.ID, upstream.TaskID),
			Status:   model.TaskStatusFailure,
			Progress: taskcommon.ProgressComplete,
			Reason:   upstream.Error.Message,
		}, nil
	}

	status := strings.ToLower(strings.TrimSpace(upstream.Status))
	if status != "queued" && status != "pending" && status != "processing" && status != "in_progress" &&
		status != "completed" && status != "failed" && status != "cancelled" {
		return nil, fmt.Errorf("wxart video task returned invalid status %q", upstream.Status)
	}

	taskID := firstNonEmpty(upstream.ID, upstream.TaskID)
	if taskID == "" && status != "failed" && status != "cancelled" {
		return nil, fmt.Errorf("wxart video task id is empty")
	}

	result := &relaycommon.TaskInfo{Code: 0, TaskID: taskID}
	switch status {
	case "queued", "pending":
		result.Status = model.TaskStatusQueued
		result.Progress = taskcommon.ProgressQueued
	case "processing", "in_progress":
		result.Status = model.TaskStatusInProgress
		result.Progress = taskcommon.ProgressInProgress
	case "completed":
		result.Status = model.TaskStatusSuccess
		result.Progress = taskcommon.ProgressComplete
		result.Url = firstNonEmpty(upstream.URL, upstream.VideoURL)
	case "failed", "cancelled":
		result.Status = model.TaskStatusFailure
		result.Progress = taskcommon.ProgressComplete
		if upstream.Error != nil && upstream.Error.Message != "" {
			result.Reason = upstream.Error.Message
		} else {
			result.Reason = "wxart video generation failed"
		}
	default:
		result.Status = model.TaskStatusInProgress
		result.Progress = taskcommon.ProgressSubmitted
	}
	if upstream.Progress > 0 && upstream.Progress < 100 {
		result.Progress = fmt.Sprintf("%d%%", upstream.Progress)
	}
	return result, nil
}

func isActiveStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return s == "queued" || s == "pending" || s == "processing" || s == "in_progress"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func mapWxartStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "processing", "in_progress":
		return dto.VideoStatusInProgress
	case "completed":
		return dto.VideoStatusCompleted
	case "failed", "cancelled":
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
		Object      string `json:"object,omitempty"`
		CreatedAt   int64  `json:"created_at,omitempty"`
		CompletedAt int64  `json:"completed_at,omitempty"`
		Status      string `json:"status,omitempty"`
		Progress    int    `json:"progress,omitempty"`
	}{
		Object:      upstream.Object,
		CreatedAt:   upstream.CreatedAt,
		CompletedAt: upstream.CompletedAt,
		Status:      upstream.Status,
		Progress:    upstream.Progress,
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
