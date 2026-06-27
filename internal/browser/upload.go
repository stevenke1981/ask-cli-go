package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AllowedImageExtensions lists supported image upload formats.
var AllowedImageExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".gif":  true,
}

// ValidateImagePath checks that the image file exists and has an allowed extension.
func ValidateImagePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("image file not found: %s", path)
		}
		return fmt.Errorf("cannot access image file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if !AllowedImageExtensions[ext] {
		return fmt.Errorf("unsupported image extension: %s (supported: png, jpg, jpeg, webp, gif)", ext)
	}

	return nil
}

// ValidateImagePaths validates all image paths and returns the first error.
func ValidateImagePaths(paths []string) error {
	for _, p := range paths {
		if err := ValidateImagePath(p); err != nil {
			return err
		}
	}
	return nil
}
