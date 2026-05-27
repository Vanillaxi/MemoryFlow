package storage

import (
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
		return "", fmt.Errorf("file is nil")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}

	//go可以根据你的模版来判断格式
	dateDir := time.Now().Format("2006/01/02")
	//如果中间目录不存在也会一起创建，0755:读写、执行权限
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

	//前端可访问的URL，不是本地文件路径
	publicURL := "/uploads/" + dateDir + "/" + filename

	return publicURL, nil
}
