package upload

import (
	"io"
	"os"
	"path/filepath"

	"github.com/siherrmann/queuerManager/helper"
)

// FilesystemLocal implements the Filesystem interface for local file storage
type FilesystemLocal struct {
	basePath string
}

// NewFilesystemLocal creates a new local filesystem instance with the specified base path
func NewFilesystemLocal(basePath string) Filesystem {
	return &FilesystemLocal{
		basePath: basePath,
	}
}

// Write streams data from reader to a file at the specified path relative to the base path
func (fs *FilesystemLocal) Write(path string, reader io.Reader, size int64) error {
	fullPath := filepath.Join(fs.basePath, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	// #nosec G304 -- Accepting file path from env variable is intentional and controlled.
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

// Open opens a file at the specified path and returns a ReadCloser
func (fs *FilesystemLocal) Open(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(fs.basePath, path)
	// #nosec G304 -- Accepting file path from variable is intentional and controlled.
	return os.Open(fullPath)
}

// Delete removes the file at the specified path
func (fs *FilesystemLocal) Delete(path string) error {
	fullPath := filepath.Join(fs.basePath, path)
	// #nosec G304 -- Accepting file path from variable is intentional and controlled.
	return os.Remove(fullPath)
}

// ListFiles returns a list of all files in the base path
func (fs *FilesystemLocal) ListFiles() ([]File, error) {
	var files []File

	// 1. Resolve the symlink if the base path itself is one (e.g. ./uploads -> /app/uploads)
	realBasePath, err := filepath.EvalSymlinks(fs.basePath)
	if err != nil {
		return nil, err
	}

	// 2. Walk the resolved path (which is now a real directory)
	err = filepath.Walk(realBasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// 3. Calculate relative path based on the resolved root
			relPath, err := filepath.Rel(realBasePath, path)
			if err != nil {
				return err
			}
			files = append(files, File{
				Name:     relPath,
				Size:     info.Size(),
				MimeType: helper.GetMimeType(relPath),
			})
		}
		return nil
	})

	return files, err
}
