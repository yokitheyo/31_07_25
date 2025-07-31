package service

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxFileSize = 20 * 1024 * 1024 // 20 MB

func DownloadAndArchive(urls []string, allowedExt []string, archivePath string) ([]string, error) {
	zipFile, err := os.Create(archivePath)
	if err != nil {
		return nil, err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	var failed []string
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (network error: %v)", url, err))
			continue
		}

		if resp.StatusCode != 200 {
			failed = append(failed, fmt.Sprintf("%s (HTTP %d)", url, resp.StatusCode))
			resp.Body.Close()
			continue
		}

		contentType := resp.Header.Get("Content-Type")
		if !isValidContentType(contentType) {
			failed = append(failed, fmt.Sprintf("%s (invalid type: %s)", url, contentType))
			resp.Body.Close()
			continue
		}

		if resp.ContentLength > maxFileSize {
			failed = append(failed, fmt.Sprintf("%s (file too large: %d bytes)", url, resp.ContentLength))
			resp.Body.Close()
			continue
		}

		fileName := generateFileName(url, contentType)

		w, err := zipWriter.Create(fileName)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (zip error: %v)", url, err))
			resp.Body.Close()
			continue
		}

		lr := &io.LimitedReader{R: resp.Body, N: maxFileSize + 1}
		_, err = io.Copy(w, lr)
		resp.Body.Close()

		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (copy error: %v)", url, err))
			continue
		}

		if lr.N <= 0 {
			failed = append(failed, fmt.Sprintf("%s (file too large)", url))
			continue
		}
	}
	return failed, nil
}

func isValidContentType(contentType string) bool {
	allowedTypes := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/jpg":       true,
	}

	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(strings.ToLower(mainType))

	return allowedTypes[mainType]
}

func generateFileName(url, contentType string) string {
	baseName := filepath.Base(url)

	if strings.Contains(baseName, "?") {
		baseName = strings.Split(baseName, "?")[0]
	}

	ext := filepath.Ext(baseName)
	hasValidExt := false

	switch strings.ToLower(contentType) {
	case "application/pdf":
		if ext != ".pdf" {
			ext = ".pdf"
		} else {
			hasValidExt = true
		}
	case "image/jpeg", "image/jpg":
		if ext != ".jpeg" && ext != ".jpg" {
			ext = ".jpeg"
		} else {
			hasValidExt = true
		}
	}

	if !hasValidExt {
		if baseName == "" || baseName == "/" || baseName == "." {
			baseName = "file"
		}
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ext
	}

	if baseName == "" {
		switch strings.ToLower(contentType) {
		case "application/pdf":
			baseName = "document.pdf"
		case "image/jpeg", "image/jpg":
			baseName = "image.jpeg"
		default:
			baseName = "file.unknown"
		}
	}

	return baseName
}
