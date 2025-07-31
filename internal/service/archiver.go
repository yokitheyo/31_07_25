package service

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// TODO: функция скачивания файла, фильтрации по расширению, упаковки в zip

const maxFileSize = 20 * 1024 * 1024 // 20 MB

// DownloadAndArchive скачивает файлы по ссылкам, фильтрует по расширению, архивирует в zip.
// Возвращает список неудачных ссылок.
func DownloadAndArchive(urls []string, allowedExt []string, archivePath string) ([]string, error) {
	zipFile, err := os.Create(archivePath)
	if err != nil {
		return nil, err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	var failed []string
	allowed := make(map[string]struct{})
	for _, ext := range allowedExt {
		allowed[ext] = struct{}{}
	}

	for _, url := range urls {
		ext := strings.ToLower(filepath.Ext(url))
		if _, ok := allowed[ext]; !ok {
			failed = append(failed, fmt.Sprintf("%s (not allowed)", url))
			continue
		}
		resp, err := http.Get(url)
		if err != nil || resp.StatusCode != 200 {
			failed = append(failed, fmt.Sprintf("%s (download error)", url))
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		if resp.ContentLength > maxFileSize {
			failed = append(failed, fmt.Sprintf("%s (file too large)", url))
			resp.Body.Close()
			continue
		}
		name := filepath.Base(url)
		w, err := zipWriter.Create(name)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (zip error)", url))
			resp.Body.Close()
			continue
		}
		lr := &io.LimitedReader{R: resp.Body, N: maxFileSize + 1}
		_, err = io.Copy(w, lr)
		resp.Body.Close()
		if err != nil || lr.N <= 0 {
			failed = append(failed, fmt.Sprintf("%s (copy error or file too large)", url))
			continue
		}
	}
	return failed, nil
}
