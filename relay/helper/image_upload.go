package helper

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	maxImageEditFiles            = 16
	maxImageEditMaskFiles        = 1
	maxImageEditTotalInputBytes  = 50 << 20
	maxImageEditTotalOutputBytes = 50 << 20
	maxImageEditTotalPixels      = 64 * 1024 * 1024
	maxImageEditStoredFiles      = 32
	maxImageEditStoredBytes      = 100 << 20
)

var imageEditUserLocks [64]sync.Mutex

func normalizeAndStoreImageEditUploads(c *gin.Context, form *multipart.Form) error {
	if form == nil || len(form.File) == 0 {
		return fmt.Errorf("image is required")
	}
	fieldNames := make([]string, 0, len(form.File))
	for fieldName := range form.File {
		if fieldName == "image" || fieldName == "image[]" || strings.HasPrefix(fieldName, "image[") || fieldName == "mask" {
			fieldNames = append(fieldNames, fieldName)
		}
	}
	sort.Strings(fieldNames)

	imageCount := 0
	maskCount := 0
	totalInputBytes := int64(0)
	totalPixels := int64(0)
	for _, fieldName := range fieldNames {
		for _, fileHeader := range form.File[fieldName] {
			if err := c.Request.Context().Err(); err != nil {
				return err
			}
			if fileHeader.Size <= 0 || fileHeader.Size > service.MaxUploadedImageBytes {
				return fmt.Errorf("%s image size must be between 1 and %d bytes", fieldName, service.MaxUploadedImageBytes)
			}
			totalInputBytes += fileHeader.Size
			if totalInputBytes > maxImageEditTotalInputBytes {
				return fmt.Errorf("image edit uploads exceed %d bytes", maxImageEditTotalInputBytes)
			}
			file, err := fileHeader.Open()
			if err != nil {
				return fmt.Errorf("open image for validation: %w", err)
			}
			metadata, inspectErr := service.InspectUploadedImage(file)
			_ = file.Close()
			if inspectErr != nil {
				return fmt.Errorf("invalid image upload in %s: %w", fieldName, inspectErr)
			}
			totalPixels += int64(metadata.Width) * int64(metadata.Height)
			if totalPixels > maxImageEditTotalPixels {
				return fmt.Errorf("image edit uploads exceed %d total pixels", maxImageEditTotalPixels)
			}
			if fieldName == "mask" {
				maskCount++
				if maskCount > maxImageEditMaskFiles {
					return fmt.Errorf("only one mask image is allowed")
				}
				continue
			}
			imageCount++
			if imageCount > maxImageEditFiles {
				return fmt.Errorf("at most %d edit images are allowed", maxImageEditFiles)
			}
		}
	}
	if imageCount == 0 {
		return fmt.Errorf("image is required")
	}

	userID := c.GetInt("id")
	lock := &imageEditUserLocks[uint(userID)%uint(len(imageEditUserLocks))]
	lock.Lock()
	defer lock.Unlock()
	directory := filepath.Join(model.UploadedImageStorageDir(), "edits", strconv.Itoa(userID))
	storageAvailable, err := service.UploadedImageStorageAvailable(directory, totalInputBytes+maxImageEditTotalOutputBytes)
	if err != nil {
		common.SysError("inspect uploaded image storage: " + err.Error())
		return fmt.Errorf("image upload storage is unavailable")
	}
	if !storageAvailable {
		return fmt.Errorf("temporary image upload quota exceeded")
	}
	if err := service.CleanupExpiredUploadedImages(directory, time.Now().Add(-service.UploadedImageRetention), 100); err != nil {
		common.SysError("cleanup expired uploaded images: " + err.Error())
	}
	existingFiles, existingBytes, err := uploadedImageDirectoryUsage(directory)
	if err != nil {
		common.SysError("inspect uploaded image directory: " + err.Error())
		return fmt.Errorf("image upload storage is unavailable")
	}
	if existingFiles+imageCount+maskCount > maxImageEditStoredFiles || existingBytes+totalInputBytes > maxImageEditStoredBytes {
		return fmt.Errorf("temporary image upload quota exceeded")
	}

	storedImages := make([]service.StoredUploadedImage, 0)
	totalOutputBytes := int64(0)
	for _, fieldName := range fieldNames {
		for index, fileHeader := range form.File[fieldName] {
			file, err := fileHeader.Open()
			if err != nil {
				removeStoredUploadedImages(storedImages)
				return fmt.Errorf("open %s image %d: %w", fieldName, index+1, err)
			}
			prefix := "edit"
			if fieldName == "mask" {
				prefix = "mask"
			}
			stored, normalizeErr := service.NormalizeAndStoreUploadedImageContext(
				c.Request.Context(), file, directory, prefix, service.MaxUploadedImageBytes, service.MaxNormalizedUploadedImageBytes,
			)
			_ = file.Close()
			if normalizeErr != nil {
				removeStoredUploadedImages(storedImages)
				if service.IsInvalidUploadedImage(normalizeErr) {
					return fmt.Errorf("invalid image upload in %s file %d: %w", fieldName, index+1, normalizeErr)
				}
				if errors.Is(normalizeErr, context.Canceled) || errors.Is(normalizeErr, context.DeadlineExceeded) {
					return normalizeErr
				}
				common.SysError(fmt.Sprintf("store normalized image upload: %v", normalizeErr))
				return fmt.Errorf("image upload storage is unavailable")
			}
			stored.FieldName = fieldName
			storedImages = append(storedImages, *stored)
			totalOutputBytes += stored.Size
			if totalOutputBytes > maxImageEditTotalOutputBytes || existingBytes+totalOutputBytes > maxImageEditStoredBytes {
				removeStoredUploadedImages(storedImages)
				return fmt.Errorf("normalized image uploads exceed storage limit")
			}
		}
	}
	c.Set(service.NormalizedImageUploadsContextKey, storedImages)
	return nil
}

func CleanupNormalizedImageUploads(c *gin.Context) {
	if c == nil {
		return
	}
	value, exists := c.Get(service.NormalizedImageUploadsContextKey)
	if exists {
		images, ok := value.([]service.StoredUploadedImage)
		if ok {
			removeStoredUploadedImages(images)
		}
		c.Set(service.NormalizedImageUploadsContextKey, nil)
	}
	if c.Request != nil && c.Request.MultipartForm != nil {
		_ = c.Request.MultipartForm.RemoveAll()
		c.Request.MultipartForm = nil
	}
}

func uploadedImageDirectoryUsage(directory string) (int, int64, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	count := 0
	totalBytes := int64(0)
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return 0, 0, infoErr
		}
		count++
		totalBytes += info.Size()
	}
	return count, totalBytes, nil
}

func removeStoredUploadedImages(images []service.StoredUploadedImage) {
	for _, image := range images {
		if err := os.Remove(image.Path); err != nil && !os.IsNotExist(err) {
			common.SysError("remove normalized image upload: " + err.Error())
		}
	}
}
