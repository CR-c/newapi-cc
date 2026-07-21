//go:build windows

package service

func UploadedImageStorageAvailable(_ string, _ int64) (bool, error) {
	return true, nil
}
