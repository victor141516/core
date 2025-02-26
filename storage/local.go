package storage

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/staticbackendhq/core/internal"
)

type Local struct{}

func (Local) Save(data internal.UploadFileData) (string, error) {
	dir := path.Join(os.TempDir(), path.Dir(data.FileKey))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	b, err := io.ReadAll(data.File)
	if err != nil {
		return "", err
	}

	filename := path.Join(os.TempDir(), data.FileKey)
	if err := os.WriteFile(filename, b, 0644); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/tmp/%s", os.Getenv("LOCAL_STORAGE_URL"), data.FileKey)
	return url, nil
}

func (Local) Delete(fileKey string) error {
	filename := path.Join(os.TempDir(), fileKey)
	return os.Remove(filename)
}
