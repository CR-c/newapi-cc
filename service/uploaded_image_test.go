package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndStoreUploadedImageNormalizesPalettedPNG(t *testing.T) {
	palette := color.Palette{color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}}
	source := image.NewPaletted(image.Rect(0, 0, 320, 320), palette)
	source.SetColorIndex(10, 10, 1)
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, source))

	stored, err := NormalizeAndStoreUploadedImage(&encoded, t.TempDir(), "edit", 10<<20, 20<<20)

	require.NoError(t, err)
	assert.Equal(t, "image/png", stored.ContentType)
	assert.Equal(t, ".png", stored.Extension)
	file, err := os.Open(stored.Path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	decoded, format, err := image.Decode(file)
	require.NoError(t, err)
	assert.Equal(t, "png", format)
	assert.Equal(t, source.Bounds(), decoded.Bounds())
	_, paletted := decoded.(*image.Paletted)
	assert.False(t, paletted)
}

func TestNormalizeAndStoreUploadedImageRejectsCorruptImage(t *testing.T) {
	dir := t.TempDir()

	_, err := NormalizeAndStoreUploadedImage(bytes.NewReader([]byte("\x89PNG\r\n\x1a\ntruncated")), dir, "edit", 10<<20, 20<<20)

	require.Error(t, err)
	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

func TestNormalizeAndStoreUploadedImageUsesSeparateOutputLimit(t *testing.T) {
	input := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			input.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 3), G: uint8(y * 5), B: uint8(x*y + y), A: 255})
		}
	}
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, input))

	stored, err := NormalizeAndStoreUploadedImage(&encoded, t.TempDir(), "edit", int64(encoded.Len()), 1<<20)

	require.NoError(t, err)
	assert.Greater(t, stored.Size, int64(0))
}

func TestCleanupExpiredUploadedImagesRemovesStaleTemporaryFiles(t *testing.T) {
	directory := t.TempDir()
	temporaryPath := filepath.Join(directory, ".image-input-stale.tmp")
	require.NoError(t, os.WriteFile(temporaryPath, []byte("stale"), 0o640))
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(temporaryPath, old, old))

	require.NoError(t, CleanupExpiredUploadedImages(directory, time.Now().Add(-time.Hour), 10))

	_, err := os.Stat(temporaryPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}
