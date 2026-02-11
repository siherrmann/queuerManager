package upload

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/siherrmann/queuerManager/helper"
)

const (
	STORAGE_MODE_LOCAL  = "local"
	STORAGE_MODE_S3     = "s3"
	STORAGE_MODE_MEMORY = "memory"
)

type File struct {
	Name     string
	Size     int64
	MimeType string
}

// Filesystem extends billy.Filesystem with additional utility methods
type Filesystem interface {
	billy.Filesystem
	Write(path string, reader io.Reader, size int64) error
	ListFiles() ([]File, error)
}

// CreateFilesystemFromEnv creates a filesystem based on environment variables
func CreateFilesystemFromEnv() (Filesystem, error) {
	storageMode := strings.ToLower(helper.GetEnvOrDefault("QUEUER_MANAGER_STORAGE_MODE", STORAGE_MODE_LOCAL))

	switch storageMode {
	case STORAGE_MODE_S3:
		config := S3Config{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Region:          helper.GetEnvOrDefault("S3_REGION", "us-east-1"),
			BucketName:      os.Getenv("S3_BUCKET_NAME"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
			UseSSL:          helper.GetEnvOrDefault("S3_USE_SSL", "true") == "true",
		}
		if config.BucketName == "" || config.AccessKeyID == "" || config.SecretAccessKey == "" {
			return nil, fmt.Errorf("missing required S3 configuration: S3_BUCKET_NAME, S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY")
		}
		return NewFilesystemS3(config)
	case STORAGE_MODE_MEMORY:
		return NewFilesystemMemory(), nil
	case STORAGE_MODE_LOCAL:
		basePath := helper.GetEnvOrDefault("QUEUER_MANAGER_STORAGE_PATH", "./uploads")
		return NewFilesystemLocal(basePath), nil
	default:
		return nil, fmt.Errorf("unsupported storage mode: %s (supported: local, s3, memory)", storageMode)
	}
}
