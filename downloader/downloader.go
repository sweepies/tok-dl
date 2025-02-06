package downloader

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/sweepies/tok-dl/util"
)

var (
	ErrDownload = errors.New("download failed")

	// Common MIME types for TikTok media
	mimeExts = map[string]string{
		"video/mp4":  ".mp4",
		"audio/mpeg": ".mp3",
		"audio/mp3":  ".mp3",
	}
)

// MediaDownloader handles downloading media files with proper error handling
type MediaDownloader struct {
	client *http.Client
}

// NewMediaDownloader creates a new media downloader
func NewMediaDownloader() *MediaDownloader {
	return &MediaDownloader{
		client: &http.Client{},
	}
}

// DownloadMedia downloads either a single video or multiple images
func (d *MediaDownloader) DownloadMedia(urls []string, destDir string) error {
	if len(urls) == 0 {
		return fmt.Errorf("no media URLs provided")
	}

	var errs []error
	for _, url := range urls {
		if err := d.downloadFile(url, destDir); err != nil {
			errs = append(errs, fmt.Errorf("failed to download %s: %w", url, err))
		}
	}

	if len(errs) > 0 {
		// If any downloads failed, return an error with details
		return fmt.Errorf("some downloads failed: %v", errs)
	}

	return nil
}

// downloadFile downloads a single file and infers its extension from content type
func (d *MediaDownloader) downloadFile(downloadURL string, destDir string) error {
	resp, err := d.client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("%w: request failed: %v", ErrDownload, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: server returned %s", ErrDownload, resp.Status)
	}

	fileName, err := generateFileName(downloadURL, resp.Header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDownload, err)
	}

	filePath := path.Join(destDir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("%w: failed to create file: %v", ErrDownload, err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("%w: failed to write file: %v", ErrDownload, err)
	}

	return nil
}

// generateFileName creates a filename from the URL and content type
func generateFileName(downloadURL string, contentType string) (string, error) {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	fileName := util.SanitizeFileName(filepath.Base(parsedURL.Path))
	if filepath.Ext(fileName) != "" {
		return fileName, nil
	}

	// If no extension in URL, infer from content type
	var ext string
	if e, ok := mimeExts[contentType]; ok {
		ext = e
	} else {
		exts, err := mime.ExtensionsByType(contentType)
		if err != nil || len(exts) == 0 {
			ext = ".bin" // fallback
		} else {
			ext = exts[0]
		}
	}

	return fmt.Sprintf("%s%s", fileName, ext), nil
}

// Close cleans up any resources
func (d *MediaDownloader) Close() error {
	d.client.CloseIdleConnections()
	return nil
}