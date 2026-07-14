package controller

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type playgroundAssetKindRule struct {
	maxSize      int64
	contentTypes map[string]string
}

var playgroundAssetKindRules = map[string]playgroundAssetKindRule{
	"image": {
		maxSize: 10 << 20,
		contentTypes: map[string]string{
			"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp",
		},
	},
	"video": {
		maxSize: 50 << 20,
		contentTypes: map[string]string{
			"video/mp4": ".mp4", "video/quicktime": ".mov", "video/webm": ".webm",
		},
	},
	"audio": {
		maxSize: 15 << 20,
		contentTypes: map[string]string{
			"audio/mpeg": ".mp3", "audio/mp4": ".m4a", "audio/x-m4a": ".m4a", "audio/wav": ".wav",
			"audio/x-wav": ".wav", "audio/aac": ".aac", "audio/ogg": ".ogg", "application/ogg": ".ogg",
		},
	},
}

const playgroundAssetUploadMaxBytes = (50 << 20) + (1 << 20)

var playgroundAssetUserLocks [64]sync.Mutex

func UploadPlaygroundAsset(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, playgroundAssetUploadMaxBytes)
	kind := strings.ToLower(strings.TrimSpace(c.PostForm("kind")))
	rule, ok := playgroundAssetKindRules[kind]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid asset kind"})
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "file is required"})
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > rule.maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": fmt.Sprintf("file size must be between 1 and %d bytes", rule.maxSize)})
		return
	}
	userID := c.GetInt("id")
	lock := &playgroundAssetUserLocks[uint(userID)%uint(len(playgroundAssetUserLocks))]
	lock.Lock()
	defer lock.Unlock()

	now := time.Now().Unix()
	assetCount, assetBytes, err := model.GetPlaygroundAssetUsage(model.DB, userID, now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if assetCount >= model.PlaygroundAssetMaxItemsPerUser || assetBytes+fileHeader.Size > model.PlaygroundAssetMaxBytesPerUser {
		c.JSON(http.StatusTooManyRequests, gin.H{"success": false, "message": "temporary asset quota exceeded"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer file.Close()
	prefix := make([]byte, 512)
	prefixSize, readErr := io.ReadFull(file, prefix)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		common.ApiError(c, readErr)
		return
	}
	prefix = prefix[:prefixSize]
	contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(http.DetectContentType(prefix), ";", 2)[0]))
	extension, ok := rule.contentTypes[contentType]
	if !ok && contentType == "application/octet-stream" && hasPlaygroundAssetSignature(kind, prefix) {
		headerType := strings.ToLower(strings.TrimSpace(strings.SplitN(fileHeader.Header.Get("Content-Type"), ";", 2)[0]))
		extension, ok = rule.contentTypes[headerType]
		contentType = headerType
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "unsupported asset content type"})
		return
	}

	assetID, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	storageName := assetID + extension
	if err = os.MkdirAll(model.PlaygroundAssetStorageDir(), 0o750); err != nil {
		common.ApiError(c, err)
		return
	}
	storagePath := model.PlaygroundAssetStoragePath(storageName)
	output, err := os.OpenFile(storagePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	written, writeErr := output.Write(prefix)
	if writeErr == nil {
		var copied int64
		copied, writeErr = io.Copy(output, io.LimitReader(file, rule.maxSize-int64(written)+1))
		written += int(copied)
	}
	closeErr := output.Close()
	if writeErr != nil || closeErr != nil || int64(written) > rule.maxSize {
		_ = os.Remove(storagePath)
		if writeErr == nil {
			writeErr = closeErr
		}
		if writeErr == nil {
			writeErr = fmt.Errorf("asset exceeds size limit")
		}
		common.ApiError(c, writeErr)
		return
	}

	filename := safePlaygroundAssetFilename(fileHeader.Filename, extension)
	asset := &model.PlaygroundAsset{
		ID: assetID, UserID: userID, Kind: kind, Filename: filename, ContentType: contentType,
		StorageName: storageName, Size: int64(written), CreatedAt: now, ExpiresAt: now + model.PlaygroundAssetTTLSeconds,
	}
	if err = model.DB.Create(asset).Error; err != nil {
		_ = os.Remove(storagePath)
		common.ApiError(c, err)
		return
	}
	_ = model.CleanupExpiredPlaygroundAssets(model.DB, now, 100)
	common.ApiSuccess(c, gin.H{
		"id": asset.ID, "kind": asset.Kind, "url": playgroundAssetURL(c, asset), "filename": asset.Filename,
		"content_type": asset.ContentType, "size": asset.Size, "expires_at": asset.ExpiresAt,
	})
}

func hasPlaygroundAssetSignature(kind string, prefix []byte) bool {
	switch kind {
	case "video":
		return (len(prefix) >= 12 && string(prefix[4:8]) == "ftyp") ||
			bytes.HasPrefix(prefix, []byte{0x1a, 0x45, 0xdf, 0xa3})
	case "audio":
		return bytes.HasPrefix(prefix, []byte("ID3")) ||
			(len(prefix) >= 2 && prefix[0] == 0xff && prefix[1]&0xe0 == 0xe0) ||
			(len(prefix) >= 12 && bytes.HasPrefix(prefix, []byte("RIFF")) && string(prefix[8:12]) == "WAVE") ||
			(len(prefix) >= 12 && string(prefix[4:8]) == "ftyp") ||
			bytes.HasPrefix(prefix, []byte("OggS"))
	default:
		return false
	}
}

func GetPlaygroundAsset(c *gin.Context) {
	now := time.Now().Unix()
	_ = model.CleanupExpiredPlaygroundAssets(model.DB, now, 100)
	asset, exists, err := model.GetPlaygroundAsset(model.DB, c.Param("asset_id"), now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !exists || asset == nil || c.Param("filename") != asset.Filename {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	path := model.PlaygroundAssetStoragePath(asset.StorageName)
	if _, err = os.Stat(path); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	remaining := asset.ExpiresAt - now
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", remaining))
	c.Header("Content-Type", asset.ContentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": asset.Filename}))
	http.ServeFile(c.Writer, c.Request, path)
}

func safePlaygroundAssetFilename(filename string, extension string) string {
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	cleaned := strings.Map(func(value rune) rune {
		if unicode.IsControl(value) || value == '/' || value == '\\' {
			return -1
		}
		return value
	}, filename)
	if cleaned == "" || cleaned == "." {
		return "asset" + extension
	}
	cleaned = strings.TrimSuffix(cleaned, filepath.Ext(cleaned))
	if cleaned == "" || cleaned == "." {
		cleaned = "asset"
	}
	runes := []rune(cleaned)
	maxBaseRunes := 180 - len([]rune(extension))
	if len(runes) > maxBaseRunes {
		cleaned = string(runes[:maxBaseRunes])
	}
	return cleaned + extension
}

func playgroundAssetURL(c *gin.Context, asset *model.PlaygroundAsset) string {
	baseURL := ""
	if trustedURLs := os.Getenv("SESSION_COOKIE_TRUSTED_URL"); trustedURLs != "" {
		baseURL = strings.TrimSpace(strings.Split(trustedURLs, ",")[0])
	}
	parsedBase, err := url.Parse(baseURL)
	if err != nil || parsedBase.Scheme == "" || parsedBase.Host == "" {
		scheme := c.GetHeader("X-Forwarded-Proto")
		if scheme != "http" && scheme != "https" {
			if c.Request.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}
		parsedBase = &url.URL{Scheme: scheme, Host: c.Request.Host}
	}
	parsedBase.Path = "/pg/assets/" + asset.ID + "/" + asset.Filename
	parsedBase.RawQuery = ""
	parsedBase.Fragment = ""
	return parsedBase.String()
}
