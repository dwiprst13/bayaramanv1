package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type StorageService interface {
	UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, folder string, filename string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}

type localStorageService struct {
	basePath string
	baseURL  string
}

func NewLocalStorageService(basePath string, baseURL string) StorageService {
	return &localStorageService{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

func (s *localStorageService) UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, folder string, filename string) (string, error) {
	const maxUploadSize = 250 << 20 // 250 MB
	if fileHeader.Size > maxUploadSize {
		return "", errors.New("file size exceeds 250MB limit")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	destFolder := filepath.Join(s.basePath, folder)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return "", err
	}

	destPath := filepath.Join(destFolder, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", s.baseURL, folder, filename), nil
}

func (s *localStorageService) DeleteFile(ctx context.Context, fileURL string) error {
	// fileURL format: http://localhost:8080/uploads/videos/file.mp4
	// baseURL format: http://localhost:8080/uploads
	// So relative path becomes: /videos/file.mp4
	if len(fileURL) <= len(s.baseURL) {
		return errors.New("invalid file url")
	}
	
	relativePath := fileURL[len(s.baseURL):]
	// Clean leading slash
	if len(relativePath) > 0 && relativePath[0] == '/' {
		relativePath = relativePath[1:]
	}

	fullPath := filepath.Join(s.basePath, relativePath)
	return os.Remove(fullPath)
}
