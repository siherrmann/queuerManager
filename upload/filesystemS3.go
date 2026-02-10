package upload

import (
	"context"
	"io"
	"manager/helper"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// FilesystemS3 implements the Filesystem interface for S3-compatible storage
type FilesystemS3 struct {
	client     *s3.Client
	bucketName string
	region     string
}

// S3Config holds the configuration for S3 filesystem
type S3Config struct {
	Endpoint        string // S3 endpoint URL (for S3-compatible services)
	Region          string // AWS region
	BucketName      string // S3 bucket name
	AccessKeyID     string // AWS access key ID
	SecretAccessKey string // AWS secret access key
	UseSSL          bool   // Whether to use SSL/TLS
}

// NewFilesystemS3 creates a new S3 filesystem instance with the specified configuration
func NewFilesystemS3(cfg S3Config) (Filesystem, error) {
	awsConfig, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for MinIO and other S3-compatible services
		}
	})

	return &FilesystemS3{
		client:     s3Client,
		bucketName: cfg.BucketName,
		region:     cfg.Region,
	}, nil
}

// Write streams data from reader to S3 at the specified path (key)
func (fs *FilesystemS3) Write(path string, reader io.Reader, size int64) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(fs.bucketName),
		Key:    aws.String(path),
		Body:   reader,
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	_, err := fs.client.PutObject(context.Background(), input)
	return err
}

// Open downloads a file from S3 and returns a ReadCloser
func (fs *FilesystemS3) Open(path string) (io.ReadCloser, error) {
	result, err := fs.client.GetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(fs.bucketName),
			Key:    aws.String(path),
		},
	)
	if err != nil {
		return nil, err
	}

	return result.Body, nil
}

// Delete removes a file from S3
func (fs *FilesystemS3) Delete(path string) error {
	_, err := fs.client.DeleteObject(
		context.Background(),
		&s3.DeleteObjectInput{
			Bucket: aws.String(fs.bucketName),
			Key:    aws.String(path),
		},
	)
	return err
}

// ListFiles returns a list of all files in the S3 bucket
func (fs *FilesystemS3) ListFiles() ([]File, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(fs.bucketName),
	}

	var files []File
	paginator := s3.NewListObjectsV2Paginator(fs.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, err
		}

		for _, object := range page.Contents {
			if object.Key != nil {
				var size int64
				if object.Size != nil {
					size = *object.Size
				}

				files = append(files, File{
					Name:     *object.Key,
					Size:     size,
					MimeType: helper.GetMimeType(*object.Key),
				})
			}
		}
	}

	return files, nil
}
