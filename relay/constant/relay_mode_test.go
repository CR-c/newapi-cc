package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath2RelayModeRecognizesPlaygroundImageRequests(t *testing.T) {
	assert.Equal(t, RelayModeImagesGenerations, Path2RelayMode("/pg/images/generations"))
	assert.Equal(t, RelayModeImagesEdits, Path2RelayMode("/pg/images/edits"))
}
