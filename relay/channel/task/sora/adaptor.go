package sora

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

type responseTask struct {
	ID                 string `json:"id"`
	TaskID             string `json:"task_id,omitempty"` //兼容旧接口
	Object             string `json:"object"`
	Model              string `json:"model"`
	Status             string `json:"status"`
	Progress           int    `json:"progress"`
	CreatedAt          int64  `json:"created_at"`
	CompletedAt        int64  `json:"completed_at,omitempty"`
	ExpiresAt          int64  `json:"expires_at,omitempty"`
	Seconds            string `json:"seconds,omitempty"`
	Size               string `json:"size,omitempty"`
	RemixedFromVideoID string `json:"remixed_from_video_id,omitempty"`
	Error              *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type legacyResponseTask struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID     string `json:"task_id"`
		Status     string `json:"status"`
		Progress   string `json:"progress"`
		ResultURL  string `json:"result_url"`
		FailReason string `json:"fail_reason"`
	} `json:"data"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType          int
	apiKey               string
	baseURL              string
	legacyOpenAIVideoAPI bool
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
	a.legacyOpenAIVideoAPI = info.ChannelOtherSettings.LegacyOpenAIVideoAPI
}

func validateRemixRequest(c *gin.Context) *dto.TaskError {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("field prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	// 存储原始请求到 context，与 ValidateMultipartDirect 路径保持一致
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if info.Action == constant.TaskActionRemix {
		return validateRemixRequest(c)
	}
	if taskErr = relaycommon.ValidateMultipartDirect(c, info); taskErr != nil {
		return taskErr
	}
	effectiveModel := info.UpstreamModelName
	if effectiveModel == "" {
		effectiveModel = info.OriginModelName
	}
	if !a.legacyOpenAIVideoAPI {
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		}
		if err = validateSub2APIVideoRequest(req, effectiveModel); err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		}
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if err = validateGrokVideoRequest(&req, effectiveModel); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	c.Set("task_request", req)
	return nil
}

func validateSub2APIVideoRequest(req relaycommon.TaskSubmitReq, modelName string) error {
	if !isSub2APIVideoModel(modelName) {
		return nil
	}
	if req.Seconds != "" {
		seconds, err := strconv.Atoi(req.Seconds)
		if err != nil || (seconds != 5 && seconds != 10 && seconds != 15) {
			return fmt.Errorf("seconds must be one of 5, 10, 15")
		}
	}
	if req.AspectRatio != "" && req.AspectRatio != "16:9" && req.AspectRatio != "9:16" && req.AspectRatio != "1:1" {
		return fmt.Errorf("aspect_ratio must be 16:9, 9:16, or 1:1")
	}
	if len(req.Images) > 4 {
		return fmt.Errorf("this model supports at most 4 reference images")
	}
	if len(req.Videos) > 3 {
		return fmt.Errorf("this model supports at most 3 reference videos")
	}
	if len(req.Audios) > 1 {
		return fmt.Errorf("this model supports at most 1 reference audio")
	}
	for _, imageURL := range req.Images {
		if err := taskcommon.ValidateMediaURL(imageURL, false); err != nil {
			return fmt.Errorf("invalid reference image: %w", err)
		}
	}
	for _, videoURL := range req.Videos {
		if err := taskcommon.ValidateMediaURL(videoURL, false); err != nil {
			return fmt.Errorf("invalid reference video: %w", err)
		}
	}
	for _, audioURL := range req.Audios {
		if err := taskcommon.ValidateMediaURL(audioURL, false); err != nil {
			return fmt.Errorf("invalid reference audio: %w", err)
		}
	}
	return nil
}

func isSub2APIVideoModel(modelName string) bool {
	return strings.HasPrefix(modelName, "video-ds-2.0") || modelName == "as-sd2.0-fast"
}

func validateGrokVideoRequest(req *relaycommon.TaskSubmitReq, modelName string) error {
	seconds := 0
	if req.Seconds != "" {
		var err error
		seconds, err = strconv.Atoi(req.Seconds)
		if err != nil {
			return fmt.Errorf("seconds must be one of 4, 6, 8, 10, 12, 15")
		}
	}
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds == 0 {
		seconds = 4
	}
	allowedSeconds := map[int]bool{4: true, 6: true, 8: true, 10: true, 12: true, 15: true}
	if !allowedSeconds[seconds] {
		return fmt.Errorf("seconds must be one of 4, 6, 8, 10, 12, 15")
	}
	if req.AspectRatio == "" {
		req.AspectRatio = "16:9"
	}
	if req.Resolution == "" {
		req.Resolution = "720p"
	}
	if req.Resolution != "720p" && req.Resolution != "480p" {
		return fmt.Errorf("resolution must be 720p or 480p")
	}
	if len(req.Videos) > 0 || len(req.Audios) > 0 {
		return fmt.Errorf("Grok video models only support reference images")
	}

	switch modelName {
	case "grok-video-1.5":
		if len(req.Images) != 1 {
			return fmt.Errorf("grok-video-1.5 requires exactly one reference image")
		}
		if req.AspectRatio != "16:9" && req.AspectRatio != "9:16" {
			return fmt.Errorf("grok-video-1.5 aspect_ratio must be 16:9 or 9:16")
		}
	case "grok-image-video":
		if len(req.Images) > 7 {
			return fmt.Errorf("grok-image-video supports at most 7 reference images")
		}
		allowedRatios := map[string]bool{"1:1": true, "16:9": true, "9:16": true, "4:3": true, "3:4": true, "3:2": true, "2:3": true}
		if !allowedRatios[req.AspectRatio] {
			return fmt.Errorf("unsupported aspect_ratio for grok-image-video")
		}
		if len(req.Images) > 1 && seconds > 10 {
			seconds = 10
		}
	}
	for _, imageURL := range req.Images {
		if err := taskcommon.ValidateMediaURL(imageURL, true); err != nil {
			return fmt.Errorf("invalid reference image: %w", err)
		}
	}
	req.Seconds = strconv.Itoa(seconds)
	req.Duration = 0
	return nil
}

// EstimateBilling 根据用户请求的 seconds 和 size 计算 OtherRatios。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	// remix 路径的 OtherRatios 已在 ResolveOriginTask 中设置
	if info.Action == constant.TaskActionRemix {
		return nil
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds <= 0 {
		seconds = 4
	}

	size := req.Size
	if size == "" {
		size = "720x1280"
	}

	ratios := map[string]float64{
		"seconds": float64(seconds),
		"size":    1,
	}
	if size == "1792x1024" || size == "1024x1792" {
		ratios["size"] = 1.666667
	}
	return ratios
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info.Action == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", a.baseURL, info.OriginTaskID), nil
	}
	if a.legacyOpenAIVideoAPI {
		return fmt.Sprintf("%s/v1/video/generations", a.baseURL), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		if a.legacyOpenAIVideoAPI {
			req, requestErr := relaycommon.GetTaskRequest(c)
			if requestErr != nil {
				return nil, requestErr
			}
			seconds, _ := strconv.Atoi(req.Seconds)
			payload := map[string]any{
				"model":        info.UpstreamModelName,
				"prompt":       req.Prompt,
				"seconds":      seconds,
				"aspect_ratio": req.AspectRatio,
				"resolution":   req.Resolution,
			}
			if len(req.Images) > 0 {
				payload["image_urls"] = req.Images
			}
			data, marshalErr := common.Marshal(payload)
			if marshalErr != nil {
				return nil, marshalErr
			}
			return bytes.NewReader(data), nil
		}
		if isSub2APIVideoModel(info.UpstreamModelName) {
			req, requestErr := relaycommon.GetTaskRequest(c)
			if requestErr != nil {
				return nil, requestErr
			}
			payload := map[string]any{
				"model":  info.UpstreamModelName,
				"prompt": req.Prompt,
			}
			if req.Seconds != "" {
				payload["seconds"] = req.Seconds
			}
			if req.AspectRatio != "" {
				payload["aspect_ratio"] = req.AspectRatio
			}
			if len(req.Images) > 0 {
				payload["images"] = req.Images
			}
			if len(req.Videos) > 0 {
				payload["videos"] = req.Videos
			}
			if len(req.Audios) > 0 {
				payload["audios"] = req.Audios
			}
			data, marshalErr := common.Marshal(payload)
			if marshalErr != nil {
				return nil, marshalErr
			}
			return bytes.NewReader(data), nil
		}
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			bodyMap["model"] = info.UpstreamModelName
			delete(bodyMap, "group")
			if newBody, err := common.Marshal(bodyMap); err == nil {
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("model", info.UpstreamModelName)
		for key, values := range formData.Value {
			if key == "model" || key == "group" {
				continue
			}
			for _, v := range values {
				writer.WriteField(key, v)
			}
		}
		for fieldName, fileHeaders := range formData.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					// Re-open after sniffing so the full content is copied below
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}
		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return common.ReaderOnly(storage), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	if a.legacyOpenAIVideoAPI {
		var directTask responseTask
		if err := common.Unmarshal(responseBody, &directTask); err == nil && (directTask.TaskID != "" || directTask.ID != "") {
			upstreamTaskID := directTask.TaskID
			if upstreamTaskID == "" {
				upstreamTaskID = directTask.ID
			}
			directTask.ID = info.PublicTaskID
			directTask.TaskID = info.PublicTaskID
			directTask.Model = info.OriginModelName
			if directTask.Object == "" {
				directTask.Object = "video"
			}
			c.JSON(http.StatusOK, directTask)
			return upstreamTaskID, responseBody, nil
		}
		var legacyTask legacyResponseTask
		if err := common.Unmarshal(responseBody, &legacyTask); err != nil {
			taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
			return
		}
		if legacyTask.Data.TaskID == "" {
			taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
			return
		}
		progress, _ := strconv.Atoi(strings.TrimSuffix(legacyTask.Data.Progress, "%"))
		clientResponse := responseTask{
			ID:       info.PublicTaskID,
			TaskID:   info.PublicTaskID,
			Object:   "video",
			Model:    info.OriginModelName,
			Status:   legacyOpenAIStatus(legacyTask.Data.Status),
			Progress: progress,
		}
		c.JSON(http.StatusOK, clientResponse)
		return legacyTask.Data.TaskID, responseBody, nil
	}

	// Parse Sora response
	var dResp responseTask
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	upstreamID := dResp.ID
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	if upstreamID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// 使用公开 task_xxxx ID 返回给客户端
	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	c.JSON(http.StatusOK, dResp)
	return upstreamID, responseBody, nil
}

func legacyOpenAIStatus(status string) string {
	switch strings.ToUpper(status) {
	case "SUCCESS", "COMPLETED":
		return dto.VideoStatusCompleted
	case "FAILURE", "FAILED", "CANCELLED":
		return dto.VideoStatusFailed
	case "PROCESSING", "IN_PROGRESS":
		return dto.VideoStatusInProgress
	default:
		return dto.VideoStatusQueued
	}
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)
	if a.legacyOpenAIVideoAPI {
		uri = fmt.Sprintf("%s/v1/video/generations/%s", baseUrl, taskID)
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	if a.legacyOpenAIVideoAPI {
		var legacyTask legacyResponseTask
		if err := common.Unmarshal(respBody, &legacyTask); err != nil {
			return nil, errors.Wrap(err, "unmarshal legacy video task result failed")
		}

		result := &relaycommon.TaskInfo{
			Code:     0,
			TaskID:   legacyTask.Data.TaskID,
			Progress: legacyTask.Data.Progress,
		}
		switch strings.ToUpper(legacyTask.Data.Status) {
		case "QUEUED", "PENDING":
			result.Status = model.TaskStatusQueued
		case "PROCESSING", "IN_PROGRESS":
			result.Status = model.TaskStatusInProgress
		case "SUCCESS", "COMPLETED":
			result.Status = model.TaskStatusSuccess
			result.Url = legacyTask.Data.ResultURL
		case "FAILURE", "FAILED", "CANCELLED":
			result.Status = model.TaskStatusFailure
			result.Reason = legacyTask.Data.FailReason
			if result.Reason == "" {
				result.Reason = legacyTask.Message
			}
		}
		return result, nil
	}

	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	switch resTask.Status {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "completed":
		taskResult.Status = model.TaskStatusSuccess
		// Url intentionally left empty — the caller constructs the proxy URL using the public task ID
	case "failed", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if resTask.Error != nil {
			taskResult.Reason = resTask.Error.Message
		} else {
			taskResult.Reason = "task failed"
		}
	default:
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", resTask.Progress)
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var standardEnvelope map[string]any
	if err := common.Unmarshal(task.Data, &standardEnvelope); err == nil {
		if _, hasStatus := standardEnvelope["status"]; hasStatus {
			data, err := sjson.SetBytes(task.Data, "id", task.TaskID)
			if err != nil {
				return nil, errors.Wrap(err, "set id failed")
			}
			data, err = sjson.SetBytes(data, "task_id", task.TaskID)
			if err != nil {
				return nil, errors.Wrap(err, "set task_id failed")
			}
			return data, nil
		}
	}
	return common.Marshal(task.ToOpenAIVideo())
}
