package kyyvideo

import "encoding/json"

import "github.com/QuantumNous/new-api/dto"

type requestPayload struct {
	Model           string   `json:"model"`
	Prompt          string   `json:"prompt"`
	Duration        int      `json:"duration"`
	Ratio           string   `json:"ratio,omitempty"`
	Resolution      string   `json:"resolution,omitempty"`
	FirstImage      string   `json:"first_image,omitempty"`
	LastImage       string   `json:"last_image,omitempty"`
	ReferenceImages []string `json:"referenceImages,omitempty"`
	ReferenceVideos []string `json:"referenceVideos,omitempty"`
	ReferenceAudios []string `json:"referenceAudios,omitempty"`
	AutoFace        *bool    `json:"autoFace,omitempty"`
}

type taskResponse struct {
	ID             string          `json:"id"`
	Object         string          `json:"object"`
	Created        dto.IntValue    `json:"created"`
	Model          string          `json:"model"`
	Status         string          `json:"status"`
	VideoURL       string          `json:"video_url"`
	ActualDuration json.RawMessage `json:"actualDuration"`
	Amount         json.RawMessage `json:"amount"`
	Error          *string         `json:"error"`
}
