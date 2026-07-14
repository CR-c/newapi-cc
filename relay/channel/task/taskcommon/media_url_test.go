package taskcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateMediaURLRejectsInternalAssetAndUnsupportedDataURLs(t *testing.T) {
	assert.Error(t, ValidateMediaURL("asset://shared-upstream-asset", false))
	assert.Error(t, ValidateMediaURL("file:///etc/passwd", false))
	assert.Error(t, ValidateMediaURL("data:image/png;base64,aGVsbG8=", false))
	assert.Error(t, ValidateMediaURL("data:text/html;base64,aGVsbG8=", true))
	assert.NoError(t, ValidateMediaURL("data:image/png;base64,aGVsbG8=", true))
}
