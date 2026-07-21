package service

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	_ "golang.org/x/image/webp"
)

const (
	NormalizedImageUploadsContextKey = "normalized_image_uploads"
	MaxUploadedImageBytes            = 50 << 20
	MaxNormalizedUploadedImageBytes  = 50 << 20
	maxUploadedImageDimension        = 8192
	maxUploadedImagePixels           = 4096 * 4096
	UploadedImageRetention           = 24 * time.Hour
)

var (
	imageDecodeSlots           = make(chan struct{}, 2)
	errNormalizedImageTooLarge = errors.New("normalized image exceeds size limit")
)

type StoredUploadedImage struct {
	FieldName   string
	Filename    string
	ContentType string
	Extension   string
	Path        string
	Size        int64
}

type UploadedImageMetadata struct {
	Format string
	Width  int
	Height int
}

type InvalidUploadedImageError struct {
	err error
}

func (e *InvalidUploadedImageError) Error() string {
	return e.err.Error()
}

func IsInvalidUploadedImage(err error) bool {
	var invalidErr *InvalidUploadedImageError
	return errors.As(err, &invalidErr)
}

func invalidUploadedImage(format string, args ...any) error {
	return &InvalidUploadedImageError{err: fmt.Errorf(format, args...)}
}

func InspectUploadedImage(reader io.Reader) (*UploadedImageMetadata, error) {
	if reader == nil {
		return nil, invalidUploadedImage("image file is required")
	}
	config, format, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, invalidUploadedImage("decode image header: %v", err)
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return nil, invalidUploadedImage("unsupported image format %q", format)
	}
	if !validUploadedImageDimensions(config.Width, config.Height) {
		return nil, invalidUploadedImage("image dimensions must be within %dx%d and %d pixels", maxUploadedImageDimension, maxUploadedImageDimension, maxUploadedImagePixels)
	}
	return &UploadedImageMetadata{Format: format, Width: config.Width, Height: config.Height}, nil
}

type boundedImageWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *boundedImageWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, errNormalizedImageTooLarge
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	return n, err
}

func NormalizeAndStoreUploadedImage(reader io.Reader, directory, prefix string, maxInputBytes, maxOutputBytes int64) (*StoredUploadedImage, error) {
	return NormalizeAndStoreUploadedImageContext(context.Background(), reader, directory, prefix, maxInputBytes, maxOutputBytes)
}

func NormalizeAndStoreUploadedImageContext(ctx context.Context, reader io.Reader, directory, prefix string, maxInputBytes, maxOutputBytes int64) (*StoredUploadedImage, error) {
	if reader == nil {
		return nil, invalidUploadedImage("image file is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if maxInputBytes <= 0 || maxInputBytes > MaxUploadedImageBytes {
		maxInputBytes = MaxUploadedImageBytes
	}
	if maxOutputBytes <= 0 || maxOutputBytes > MaxNormalizedUploadedImageBytes {
		maxOutputBytes = MaxNormalizedUploadedImageBytes
	}

	select {
	case imageDecodeSlots <- struct{}{}:
		defer func() { <-imageDecodeSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, fmt.Errorf("create image directory: %w", err)
	}
	input, err := os.CreateTemp(directory, ".image-input-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create image input: %w", err)
	}
	inputPath := input.Name()
	defer func() {
		_ = input.Close()
		_ = os.Remove(inputPath)
	}()
	if err = input.Chmod(0o640); err != nil {
		return nil, fmt.Errorf("set image input permissions: %w", err)
	}
	written, err := io.Copy(input, io.LimitReader(reader, maxInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if written == 0 {
		return nil, invalidUploadedImage("image file is empty")
	}
	if written > maxInputBytes {
		return nil, invalidUploadedImage("image file exceeds %d bytes", maxInputBytes)
	}
	if _, err = input.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek image input: %w", err)
	}
	metadata, err := InspectUploadedImage(input)
	if err != nil {
		return nil, err
	}
	if _, err = input.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek image input: %w", err)
	}
	decoded, decodedFormat, err := image.Decode(input)
	if err != nil {
		return nil, invalidUploadedImage("decode image: %v", err)
	}
	if decodedFormat != metadata.Format {
		return nil, invalidUploadedImage("image format changed while decoding")
	}
	bounds := decoded.Bounds()
	if !validUploadedImageDimensions(bounds.Dx(), bounds.Dy()) {
		return nil, invalidUploadedImage("decoded image dimensions are invalid")
	}
	normalized := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(normalized, normalized.Bounds(), decoded, bounds.Min, draw.Src)

	output, err := os.CreateTemp(directory, ".image-output-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create image output: %w", err)
	}
	outputPath := output.Name()
	removeOutput := true
	defer func() {
		_ = output.Close()
		if removeOutput {
			_ = os.Remove(outputPath)
		}
	}()
	if err = output.Chmod(0o640); err != nil {
		return nil, fmt.Errorf("set image output permissions: %w", err)
	}
	limitedOutput := &boundedImageWriter{writer: output, remaining: maxOutputBytes}
	if err = png.Encode(limitedOutput, normalized); err != nil {
		if errors.Is(err, errNormalizedImageTooLarge) {
			return nil, invalidUploadedImage("normalized image exceeds %d bytes", maxOutputBytes)
		}
		return nil, fmt.Errorf("encode normalized image: %w", err)
	}
	if err = output.Sync(); err != nil {
		return nil, fmt.Errorf("sync image: %w", err)
	}
	info, err := output.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat image: %w", err)
	}
	if err = output.Close(); err != nil {
		return nil, fmt.Errorf("close image: %w", err)
	}

	prefix = sanitizeUploadedImagePrefix(prefix)
	key, err := common.GenerateRandomCharsKey(24)
	if err != nil {
		return nil, fmt.Errorf("generate image key: %w", err)
	}
	filename := prefix + "-" + key + ".png"
	path := filepath.Join(directory, filename)
	if err = os.Rename(outputPath, path); err != nil {
		return nil, fmt.Errorf("publish image: %w", err)
	}
	removeOutput = false

	return &StoredUploadedImage{
		Filename: filename, ContentType: "image/png", Extension: ".png", Path: path, Size: info.Size(),
	}, nil
}

func validUploadedImageDimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= maxUploadedImageDimension && height <= maxUploadedImageDimension && int64(width)*int64(height) <= maxUploadedImagePixels
}

func sanitizeUploadedImagePrefix(prefix string) string {
	prefix = filepath.Base(strings.TrimSpace(prefix))
	var cleaned strings.Builder
	for _, character := range prefix {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' || character == '_' {
			cleaned.WriteRune(character)
		}
	}
	value := strings.Trim(cleaned.String(), "-_")
	if value == "" {
		return "image"
	}
	return value
}

func CleanupExpiredUploadedImages(directory string, cutoff time.Time, limit int) error {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	dir, err := os.Open(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer dir.Close()

	removed := 0
	for removed < limit {
		entries, readErr := dir.ReadDir(100)
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			isTemporary := strings.HasPrefix(entry.Name(), ".image-") && strings.HasSuffix(entry.Name(), ".tmp")
			if strings.HasPrefix(entry.Name(), ".") && !isTemporary {
				continue
			}
			info, infoErr := entry.Info()
			if infoErr != nil || !info.ModTime().Before(cutoff) {
				continue
			}
			if removeErr := os.Remove(filepath.Join(directory, entry.Name())); removeErr != nil && !os.IsNotExist(removeErr) {
				return removeErr
			}
			removed++
			if removed == limit {
				break
			}
		}
		if readErr == io.EOF {
			break
		}
	}
	return nil
}
