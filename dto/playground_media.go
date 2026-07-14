package dto

type PlaygroundMediaHistory struct {
	ID          string      `json:"id"`
	MediaType   string      `json:"media_type"`
	Model       string      `json:"model"`
	Prompt      string      `json:"prompt,omitempty"`
	TaskID      string      `json:"task_id,omitempty"`
	Status      string      `json:"status"`
	Progress    int         `json:"progress"`
	ResultURL   string      `json:"result_url,omitempty"`
	Images      []ImageData `json:"images,omitempty"`
	Error       string      `json:"error,omitempty"`
	CreatedAt   int64       `json:"created_at"`
	CompletedAt int64       `json:"completed_at,omitempty"`
}
