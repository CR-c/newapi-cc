package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// uploadedImageCleanupInterval covers edit-upload directories.
// playgroundAssetCleanupInterval is tighter so 30-minute temp assets are swept promptly.
const (
	uploadedImageCleanupInterval   = 15 * time.Minute
	playgroundAssetCleanupInterval = 5 * time.Minute
)

func StartUploadedImageCleanupTask() {
	go func() {
		cleanupPlaygroundAssets()
		cleanupUploadedImages()
		playgroundTicker := time.NewTicker(playgroundAssetCleanupInterval)
		uploadedTicker := time.NewTicker(uploadedImageCleanupInterval)
		defer playgroundTicker.Stop()
		defer uploadedTicker.Stop()
		for {
			select {
			case <-playgroundTicker.C:
				cleanupPlaygroundAssets()
			case <-uploadedTicker.C:
				cleanupUploadedImages()
			}
		}
	}()
}

func cleanupPlaygroundAssets() {
	now := time.Now()
	deleted, err := model.CleanupExpiredPlaygroundAssets(model.DB, now.Unix(), 500)
	if err != nil {
		common.SysError("cleanup expired playground assets: " + err.Error())
	} else if deleted > 0 {
		common.SysLog(fmt.Sprintf("cleanup expired playground assets: deleted %d", deleted))
	}
	// Safety net: remove unreferenced files older than the 30-minute TTL.
	orphans, orphanErr := model.CleanupOrphanPlaygroundAssetFiles(time.Duration(model.PlaygroundAssetTTLSeconds)*time.Second, 2000)
	if orphanErr != nil {
		common.SysError("cleanup orphan playground asset files: " + orphanErr.Error())
	} else if orphans > 0 {
		common.SysLog(fmt.Sprintf("cleanup orphan playground asset files: deleted %d", orphans))
	}
}

func cleanupUploadedImages() {
	now := time.Now()
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
