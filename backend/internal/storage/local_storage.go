package storage

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalStorage struct {
	UploadDir string
}

func NewLocalStorage(uploadDir string) *LocalStorage {
	return &LocalStorage{
		UploadDir: uploadDir,
	}
}

func (s *LocalStorage) SaveImage(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", errors.New("file is nil")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}

	dateDir := time.Now().Format("2006-01-02")
	saveDir := filepath.Join(s.UploadDir, dateDir)

	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(saveDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(savePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return savePath, nil
}
