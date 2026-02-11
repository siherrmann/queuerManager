package upload

import (
	"io"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/siherrmann/queuerManager/helper"
)

// FilesystemMemory implements the Filesystem interface for in-memory file storage using go-billy's memfs
type FilesystemMemory struct {
	billy.Filesystem
}

// NewFilesystemMemory creates a new in-memory filesystem instance
func NewFilesystemMemory() Filesystem {
	return &FilesystemMemory{
		Filesystem: memfs.New(),
	}
}

// Write streams data from reader to a file at the specified path
func (fs *FilesystemMemory) Write(path string, reader io.Reader, size int64) error {
	file, err := fs.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

// ListFiles returns a list of all files in the filesystem
func (fs *FilesystemMemory) ListFiles() ([]File, error) {
	var files []File

	// Helper function to walk the directory tree
	var walk func(string) error
	walk = func(dirPath string) error {
		entries, err := fs.ReadDir(dirPath)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			entryPath := fs.Join(dirPath, entry.Name())
			if entry.IsDir() {
				if err := walk(entryPath); err != nil {
					return err
				}
			} else {
				// Get relative path for display
				relPath := entryPath
				if dirPath == "." || dirPath == "" {
					relPath = entry.Name()
				}

				files = append(files, File{
					Name:     filepath.ToSlash(relPath),
					Size:     entry.Size(),
					MimeType: helper.GetMimeType(entry.Name()),
				})
			}
		}
		return nil
	}

	if err := walk("."); err != nil {
		return nil, err
	}

	return files, nil
}
