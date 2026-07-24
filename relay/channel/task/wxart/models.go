package wxart

// requestPayload is the upstream create-task body for api.wxart.space.
// Field names differ by model:
//   - grok-imagine-video-1.5-preview: size (or resolution), images_url (exactly 1)
//   - grok-video-3: resolution only (rejects size), mode, images_url by mode
type requestPayload struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Mode        string   `json:"mode,omitempty"`
	ImagesURL   []string `json:"images_url,omitempty"`
	AspectRatio string   `json:"aspect_ratio,omitempty"`
	Duration    int      `json:"duration"`
	Size        string   `json:"size,omitempty"`
	Resolution  string   `json:"resolution,omitempty"`
}

// taskResponse matches OpenAI-style video task envelopes returned by wxart.
type taskResponse struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id,omitempty"`
	Object      string `json:"object,omitempty"`
	Model       string `json:"model,omitempty"`
	Status      string `json:"status"`
	Progress    int    `json:"progress,omitempty"`
	CreatedAt   int64  `json:"created_at,omitempty"`
	CompletedAt int64  `json:"completed_at,omitempty"`
	URL         string `json:"url,omitempty"`
	VideoURL    string `json:"video_url,omitempty"`
	Error       *struct {
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}
