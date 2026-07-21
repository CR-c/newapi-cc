package service

import (
	"os"
	"path/filepath"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const uploadedImageCleanupInterval = 15 * time.Minute

func StartUploadedImageCleanupTask() {
	go func() {
		cleanupUploadedImages()
		ticker := time.NewTicker(uploadedImageCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			cleanupUploadedImages()
		}
	}()
}

func cleanupUploadedImages() {
	now := time.Now()
	if err := model.CleanupExpiredPlaygroundAssets(model.DB, now.Unix(), 500); err != nil {
		common.SysError("cleanup expired playground assets: " + err.Error())
	}
	editsDirectory := filepath.Join(model.UploadedImageStorageDir(), "edits")
	entries, err := os.ReadDir(editsDirectory)
	if err != nil {
		if !os.IsNotExist(err) {
			common.SysError("read uploaded image cleanup directory: " + err.Error())
		}
		return
	}
	cutoff := now.Add(-UploadedImageRetention)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		directory := filepath.Join(editsDirectory, entry.Name())
		if err = CleanupExpiredUploadedImages(directory, cutoff, 500); err != nil {
			common.SysError("cleanup expired edit uploads: " + err.Error())
			continue
		}
		_ = os.Remove(directory)
	}
}
