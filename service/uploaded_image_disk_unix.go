//go:build !windows

package service

import (
	"os"

	"golang.org/x/sys/unix"
)

const uploadedImageMinimumFreeBytes = 1 << 30

func UploadedImageStorageAvailable(directory string, incomingBytes int64) (bool, error) {
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return false, err
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(directory, &stat); err != nil {
		return false, err
	}
	available := int64(stat.Bavail) * int64(stat.Bsize)
	return available-incomingBytes >= uploadedImageMinimumFreeBytes, nil
}
