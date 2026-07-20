package relay

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskSubmitSucceededAcceptsAnyTwoHundredStatus(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{status: http.StatusContinue, want: false},
		{status: http.StatusOK, want: true},
		{status: http.StatusCreated, want: true},
		{status: http.StatusAccepted, want: true},
		{status: 299, want: true},
		{status: http.StatusMultipleChoices, want: false},
		{status: http.StatusBadRequest, want: false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, taskSubmitSucceeded(tt.status))
	}
}
