package browser

import "fmt"

// Downloader handles downloading images from ChatGPT responses.
type Downloader struct {
	// OutputDir is the directory where downloaded images are saved.
	OutputDir string
}

// NewDownloader creates a new Downloader.
func NewDownloader(outputDir string) *Downloader {
	return &Downloader{OutputDir: outputDir}
}

// DownloadImages downloads images from the given source URLs.
// This is a placeholder; actual implementation will use CDP or HTTP to fetch images.
func (d *Downloader) DownloadImages(urls []string) ([]string, error) {
	if len(urls) == 0 {
		return nil, nil
	}
	// TODO: Implement actual image download
	return nil, fmt.Errorf("image download not yet implemented")
}
