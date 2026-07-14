package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPlaygroundResponseWriterDropsCaptureWhenLimitExceeded(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	writer := &playgroundResponseWriter{ResponseWriter: context.Writer, maxSize: 4}

	_, err := writer.Write([]byte("ab"))
	require.NoError(t, err)
	_, err = writer.WriteString("cd")
	require.NoError(t, err)
	assert.Equal(t, "abcd", writer.body.String())

	_, err = writer.Write([]byte("e"))
	require.NoError(t, err)
	assert.True(t, writer.truncated)
	assert.Zero(t, writer.body.Len())
	assert.Equal(t, "abcde", recorder.Body.String(), "capturing must not alter the client response")
}

func TestBuildPlaygroundImageHistoryParsesSuccessfulImageResponse(t *testing.T) {
	responseBody := []byte(`{"created":1784010015,"data":[{"url":"https://example.com/result.png","revised_prompt":"a refined prompt"}]}`)

	history, err := buildPlaygroundImageHistory(7, "gpt-image-1", "draw a city", responseBody, time.Unix(1784010015, 0))
	require.NoError(t, err)
	assert.Equal(t, 7, history.UserID)
	assert.Equal(t, model.MediaTypeImage, history.MediaType)
	assert.Equal(t, "gpt-image-1", history.Model)
	assert.Equal(t, "draw a city", history.Prompt)
	assert.Equal(t, int64(1784010015), history.CreatedAt)
	assert.Equal(t, int64(1784010015+model.PlaygroundMediaHistoryTTLSeconds), history.ExpiresAt)

	var images []dto.ImageData
	require.NoError(t, history.DecodeResult(&images))
	require.Len(t, images, 1)
	assert.Equal(t, "https://example.com/result.png", images[0].Url)
	assert.Equal(t, "a refined prompt", images[0].RevisedPrompt)
}

func TestPersistPlaygroundImageHistoryStoresAccountScopedResult(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PlaygroundMediaHistory{}))

	responseBody := []byte(`{"data":[{"url":"https://example.com/result.png"}]}`)
	require.NoError(t, persistPlaygroundImageHistory(db, 17, "gpt-image-1", "draw a city", responseBody, time.Unix(1784010015, 0)))

	items, err := model.GetPlaygroundImageHistory(db, 17, 1784010020, 20)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 17, items[0].UserID)
	assert.Equal(t, "draw a city", items[0].Prompt)
}

func TestBuildPlaygroundImageHistoryRejectsResponseWithoutResults(t *testing.T) {
	_, err := buildPlaygroundImageHistory(7, "gpt-image-1", "draw a city", []byte(`{"data":[]}`), time.Now())
	require.Error(t, err)
}

func TestBuildPlaygroundImageHistoryRejectsUnsafeOrBase64Results(t *testing.T) {
	responseBody := []byte(`{"data":[{"url":"javascript:alert(1)"},{"b64_json":"large-image-data"}]}`)

	_, err := buildPlaygroundImageHistory(7, "gpt-image-1", "draw a city", responseBody, time.Now())
	require.Error(t, err)
}

func TestPlaygroundVideoHistoryDTOUsesLocalContentURL(t *testing.T) {
	task := &model.Task{
		TaskID:      "task_public",
		Status:      model.TaskStatusSuccess,
		Progress:    "100%",
		CreatedAt:   1784010015,
		FinishTime:  1784010761,
		Properties:  model.Properties{OriginModelName: "dreamina-seedance-2-0-mini-hc"},
		PrivateData: model.TaskPrivateData{ResultURL: "https://upstream.example/private.mp4"},
	}

	item := playgroundVideoHistoryDTO(task)
	assert.Equal(t, "completed", item.Status)
	assert.Equal(t, 100, item.Progress)
	assert.Equal(t, "/v1/videos/task_public/content", item.ResultURL)
	assert.Equal(t, "dreamina-seedance-2-0-mini-hc", item.Model)
	assert.Empty(t, item.Error)
}

func TestPlaygroundVideoHistoryDTOMapsNonSuccessStates(t *testing.T) {
	tests := []struct {
		name             string
		task             *model.Task
		expectedStatus   string
		expectedProgress int
		expectedError    string
	}{
		{
			name: "queued",
			task: &model.Task{
				TaskID: "task_queued", Status: model.TaskStatusQueued, Progress: "25%", CreatedAt: 100,
			},
			expectedStatus: "queued", expectedProgress: 25,
		},
		{
			name: "failed with fallback timestamps",
			task: &model.Task{
				TaskID: "task_failed", Status: model.TaskStatusFailure, Progress: "invalid", SubmitTime: 100, UpdatedAt: 200, FailReason: "upstream failed",
			},
			expectedStatus: "failed", expectedProgress: 0, expectedError: "upstream failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := playgroundVideoHistoryDTO(test.task)
			assert.Equal(t, test.expectedStatus, item.Status)
			assert.Equal(t, test.expectedProgress, item.Progress)
			assert.Equal(t, test.expectedError, item.Error)
			assert.Empty(t, item.ResultURL)
			if test.task.CreatedAt == 0 {
				assert.Equal(t, test.task.SubmitTime, item.CreatedAt)
				assert.Equal(t, test.task.UpdatedAt, item.CompletedAt)
			}
		})
	}
}

func TestGetPlaygroundMediaHistoryReturnsOnlyCurrentUsersImages(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PlaygroundMediaHistory{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	now := time.Now()
	for _, userID := range []int{7, 8} {
		history := model.NewPlaygroundImageHistory(userID, "gpt-image-1", "private prompt", `[{"url":"https://example.com/result.png"}]`, now)
		require.NoError(t, history.Insert(db))
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/playground/media-history?media_type=image", nil)
	context.Set("id", 7)
	GetPlaygroundMediaHistory(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := struct {
		Success bool                         `json:"success"`
		Data    []dto.PlaygroundMediaHistory `json:"data"`
	}{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Equal(t, "gpt-image-1", response.Data[0].Model)
	assert.NotContains(t, recorder.Body.String(), `"user_id":8`)
}

func TestGetPlaygroundMediaHistoryRejectsInvalidType(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/playground/media-history?media_type=audio", nil)

	GetPlaygroundMediaHistory(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
