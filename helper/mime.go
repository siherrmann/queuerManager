package helper

import (
	"mime"
	"path/filepath"
)

// GetMimeType returns the MIME type for a file based on its extension
func GetMimeType(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream" // Default for unknown file types
	}
	return mimeType
}
