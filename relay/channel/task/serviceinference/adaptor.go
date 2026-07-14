package serviceinference

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type contentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
	Role string `json:"role,omitempty"`
}
type requestPayload struct {
	Model         string        `json:"model"`
	Content       []contentItem `json:"content"`
	Duration      int           `json:"duration,omitempty"`
	Resolution    string        `json:"resolution,omitempty"`
	Ratio         string        `json:"ratio,omitempty"`
	GenerateAudio *bool         `json:"generate_audio,omitempty"`
	Watermark     *bool         `json:"watermark,omitempty"`
}
type taskEnvelope struct {
	Task struct {
		ID      string   `json:"id"`
		Status  string   `json:"status"`
		Outputs []string `json:"outputs"`
		Error   any      `json:"error"`
		Usage   struct {
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	} `json:"task"`
}

type assetResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID       string `json:"Id"`
		BaseResp struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	} `json:"data"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

const (
	minDurationSeconds     = 4
	maxDurationSeconds     = 15
	defaultDurationSeconds = 4
)

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
	duration := req.Duration
	if duration == 0 && req.Seconds != "" {
		duration, err = strconv.Atoi(req.Seconds)
		if err != nil {
			return service.TaskErrorWrapperLocal(fmt.Errorf("duration must be an integer"), "invalid_duration", http.StatusBadRequest)
		}
	}
	if duration == 0 {
		duration = defaultDurationSeconds
	}
	if len(req.Images) > 4 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("at most 4 reference images are supported"), "invalid_images", http.StatusBadRequest)
	}
	if len(req.Videos) > 0 || len(req.Audios) > 0 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("this model only supports reference images"), "invalid_media", http.StatusBadRequest)
	}
	for _, imageURL := range req.Images {
		if err = taskcommon.ValidateMediaURL(imageURL, false); err != nil {
			return service.TaskErrorWrapperLocal(fmt.Errorf("invalid reference image: %w", err), "invalid_images", http.StatusBadRequest)
		}
	}
	if duration < minDurationSeconds || duration > maxDurationSeconds {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("duration must be between %d and %d seconds", minDurationSeconds, maxDurationSeconds),
			"invalid_duration",
			http.StatusBadRequest,
		)
	}
	req.Duration = duration
	req.Seconds = ""
	c.Set("task_request", req)
	return nil
}
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + "/v1/video/generate", nil
}
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	resolution := taskResolution(req)
	billingModel := info.UpstreamModelName
	if billingModel == "" {
		billingModel = info.OriginModelName
	}
	ratio, ok := billingRatio(billingModel, resolution, req.HasImage())
	if !ok || ratio == 1 {
		return nil
	}
	return map[string]float64{"reference_image": ratio}
}

func taskResolution(req relaycommon.TaskSubmitReq) string {
	if req.Resolution != "" {
		return req.Resolution
	}
	if resolution, _ := req.Metadata["resolution"].(string); resolution != "" {
		return resolution
	}
	switch strings.ToLower(req.Size) {
	case "1024x1792", "1792x1024", "1080p":
		return "1080p"
	case "720x1280", "1280x720", "720p":
		return "720p"
	case "4k":
		return "4k"
	default:
		return ""
	}
}
func billingRatio(modelName, resolution string, hasRef bool) (float64, bool) {
	base := 0.0
	price := 0.0
	switch modelName {
	case "dreamina-seedance-2-0-fast-hc":
		base = 5.6
		if hasRef {
			price = 3.3
		} else {
			price = 5.6
		}
	case "dreamina-seedance-2-0-hc":
		base = 7
		switch strings.ToLower(resolution) {
		case "4k":
			if hasRef {
				price = 2.4
			} else {
				price = 4
			}
		case "1080p":
			if hasRef {
				price = 4.7
			} else {
				price = 7.7
			}
		default:
			if hasRef {
				price = 4.3
			} else {
				price = 7
			}
		}
	case "dreamina-seedance-2-0-mini-hc":
		base = 3.5
		if hasRef {
			price = 2.1
		} else {
			price = 3.5
		}
	default:
		return 0, false
	}
	return price / base, true
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	body := requestPayload{
		Model: info.UpstreamModelName, Content: []contentItem{{Type: "text", Text: req.Prompt}}, Duration: req.Duration,
		Resolution: taskResolution(req), Ratio: req.AspectRatio, GenerateAudio: req.GenerateAudio, Watermark: req.Watermark,
	}
	if body.Model == "" {
		body.Model = req.Model
	}
	if seconds := req.Seconds; seconds != "" {
		_, _ = fmt.Sscan(seconds, &body.Duration)
	}
	if body.Ratio == "" {
		if ratio, ok := req.Metadata["ratio"].(string); ok {
			body.Ratio = ratio
		}
	}
	for _, imageURL := range req.Images {
		assetID, uploadErr := a.createImageAsset(c, info, imageURL)
		if uploadErr != nil {
			return nil, uploadErr
		}
		body.Content = append(body.Content, contentItem{Type: "image_url", ImageURL: &struct {
			URL string `json:"url"`
		}{URL: "asset://" + assetID}, Role: "reference_image"})
	}
	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) createImageAsset(c *gin.Context, info *relaycommon.RelayInfo, imageURL string) (string, error) {
	payload := map[string]string{
		"URL": imageURL, "Name": "playground-reference", "AssetType": "Image",
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	requestContext := context.Background()
	if c != nil && c.Request != nil {
		requestContext = c.Request.Context()
	}
	req, err := http.NewRequestWithContext(requestContext, http.MethodPost, a.baseURL+"/v1/sd/assets", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	proxy := ""
	if info != nil && info.ChannelMeta != nil {
		proxy = info.ChannelSetting.Proxy
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return "", err
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("asset upload failed with status %d", resp.StatusCode)
	}
	var result assetResponse
	if err = common.Unmarshal(responseBody, &result); err != nil {
		return "", err
	}
	if !result.Success || result.Data.BaseResp.StatusCode != 0 || result.Data.ID == "" {
		return "", fmt.Errorf("asset upload failed: %s", result.Data.BaseResp.StatusMsg)
	}
	return result.Data.ID, nil
}
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, body)
}
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", 500)
	}
	_ = resp.Body.Close()
	var task taskEnvelope
	if err = common.Unmarshal(b, &task); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_body_failed", 500)
	}
	if task.Task.ID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("task id is empty"), "invalid_response", 500)
	}
	out := dto.NewOpenAIVideo()
	out.ID = info.PublicTaskID
	out.TaskID = info.PublicTaskID
	out.CreatedAt = time.Now().Unix()
	out.Model = info.OriginModelName
	c.JSON(http.StatusOK, out)
	return task.Task.ID, b, nil
}
func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	id, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task id")
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/video/tasks/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}
func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	var upstream taskEnvelope
	if err := common.Unmarshal(body, &upstream); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result")
	}
	result := &relaycommon.TaskInfo{TotalTokens: upstream.Task.Usage.TotalTokens, CompletionTokens: upstream.Task.Usage.CompletionTokens}
	switch upstream.Task.Status {
	case "completed":
		result.Status = model.TaskStatusSuccess
		result.Progress = "100%"
		if len(upstream.Task.Outputs) > 0 {
			result.Url = upstream.Task.Outputs[0]
		}
	case "failed":
		result.Status = model.TaskStatusFailure
		result.Progress = "100%"
		result.Reason = fmt.Sprint(upstream.Task.Error)
	default:
		result.Status = model.TaskStatusInProgress
		result.Progress = "50%"
	}
	return result, nil
}
func (a *TaskAdaptor) GetModelList() []string {
	return []string{"dreamina-seedance-2-0-fast-hc", "dreamina-seedance-2-0-hc", "dreamina-seedance-2-0-mini-hc"}
}
func (a *TaskAdaptor) GetChannelName() string { return "service-inference" }
func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	return common.Marshal(task.ToOpenAIVideo())
}
