package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/siherrmann/queuerManager/helper"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// FilesystemS3 implements the billy.Filesystem interface for S3-compatible storage
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

// ===== billy.Filesystem interface implementation =====

// Create creates a new file for writing
func (fs *FilesystemS3) Create(filename string) (billy.File, error) {
	return &s3File{
		fs:     fs,
		path:   filename,
		buffer: &bytes.Buffer{},
		mode:   os.O_CREATE | os.O_WRONLY | os.O_TRUNC,
	}, nil
}

// Open opens a file for reading
func (fs *FilesystemS3) Open(filename string) (billy.File, error) {
	return &s3File{
		fs:   fs,
		path: filename,
		mode: os.O_RDONLY,
	}, nil
}

// OpenFile opens a file with the specified flag and permissions
func (fs *FilesystemS3) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	return &s3File{
		fs:     fs,
		path:   filename,
		buffer: &bytes.Buffer{},
		mode:   flag,
	}, nil
}

// Stat returns file info for the specified path
func (fs *FilesystemS3) Stat(filename string) (os.FileInfo, error) {
	result, err := fs.client.HeadObject(
		context.Background(),
		&s3.HeadObjectInput{
			Bucket: aws.String(fs.bucketName),
			Key:    aws.String(filename),
		},
	)
	if err != nil {
		return nil, err
	}

	return &s3FileInfo{
		name:    path.Base(filename),
		size:    *result.ContentLength,
		modTime: *result.LastModified,
		isDir:   false,
	}, nil
}

// Rename renames a file (not efficiently supported in S3)
func (fs *FilesystemS3) Rename(oldpath, newpath string) error {
	// Copy to new location
	_, err := fs.client.CopyObject(
		context.Background(),
		&s3.CopyObjectInput{
			Bucket:     aws.String(fs.bucketName),
			CopySource: aws.String(path.Join(fs.bucketName, oldpath)),
			Key:        aws.String(newpath),
		},
	)
	if err != nil {
		return err
	}

	// Delete old location
	return fs.Remove(oldpath)
}

// Remove deletes a file
func (fs *FilesystemS3) Remove(filename string) error {
	_, err := fs.client.DeleteObject(
		context.Background(),
		&s3.DeleteObjectInput{
			Bucket: aws.String(fs.bucketName),
			Key:    aws.String(filename),
		},
	)
	return err
}

// Join joins path elements
func (fs *FilesystemS3) Join(elem ...string) string {
	return path.Join(elem...)
}

// TempFile creates a temporary file (not efficiently supported in S3)
func (fs *FilesystemS3) TempFile(dir, prefix string) (billy.File, error) {
	tempName := path.Join(dir, fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()))
	return fs.Create(tempName)
}

// ReadDir lists files in a directory
func (fs *FilesystemS3) ReadDir(dirPath string) ([]os.FileInfo, error) {
	prefix := dirPath
	if prefix != "" && prefix != "." && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if prefix == "." {
		prefix = ""
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(fs.bucketName),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	result, err := fs.client.ListObjectsV2(context.Background(), input)
	if err != nil {
		return nil, err
	}

	var infos []os.FileInfo

	// Add directories (common prefixes)
	for _, commonPrefix := range result.CommonPrefixes {
		if commonPrefix.Prefix != nil {
			name := strings.TrimPrefix(*commonPrefix.Prefix, prefix)
			name = strings.TrimSuffix(name, "/")
			if name != "" {
				infos = append(infos, &s3FileInfo{
					name:  name,
					isDir: true,
				})
			}
		}
	}

	// Add files
	for _, object := range result.Contents {
		if object.Key != nil {
			name := strings.TrimPrefix(*object.Key, prefix)
			if name != "" && name != "/" {
				var size int64
				if object.Size != nil {
					size = *object.Size
				}
				var modTime time.Time
				if object.LastModified != nil {
					modTime = *object.LastModified
				}
				infos = append(infos, &s3FileInfo{
					name:    name,
					size:    size,
					modTime: modTime,
					isDir:   false,
				})
			}
		}
	}

	return infos, nil
}

// MkdirAll creates all directories in the path (no-op for S3)
func (fs *FilesystemS3) MkdirAll(filename string, perm os.FileMode) error {
	// S3 doesn't have directories, so this is a no-op
	return nil
}

// Lstat returns file info (same as Stat for S3)
func (fs *FilesystemS3) Lstat(filename string) (os.FileInfo, error) {
	return fs.Stat(filename)
}

// Symlink is not supported on S3
func (fs *FilesystemS3) Symlink(target, link string) error {
	return errors.New("symlinks not supported on S3")
}

// Readlink is not supported on S3
func (fs *FilesystemS3) Readlink(link string) (string, error) {
	return "", errors.New("symlinks not supported on S3")
}

// Chroot creates a new filesystem rooted at the given path
func (fs *FilesystemS3) Chroot(path string) (billy.Filesystem, error) {
	return nil, errors.New("chroot not supported on S3 filesystem")
}

// Root returns the root path of the filesystem
func (fs *FilesystemS3) Root() string {
	return "/"
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

// ===== Internal types for S3 =====

// s3FileInfo implements os.FileInfo for S3 objects
type s3FileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (fi *s3FileInfo) Name() string       { return fi.name }
func (fi *s3FileInfo) Size() int64        { return fi.size }
func (fi *s3FileInfo) Mode() os.FileMode  { return 0644 }
func (fi *s3FileInfo) ModTime() time.Time { return fi.modTime }
func (fi *s3FileInfo) IsDir() bool        { return fi.isDir }
func (fi *s3FileInfo) Sys() interface{}   { return nil }

// s3File implements billy.File interface for S3 objects
type s3File struct {
	fs       *FilesystemS3
	path     string
	buffer   *bytes.Buffer
	reader   io.ReadCloser
	position int64
	mode     int
	closed   bool
}

func (f *s3File) Name() string {
	return f.path
}

func (f *s3File) Write(p []byte) (n int, err error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	if f.buffer == nil {
		f.buffer = &bytes.Buffer{}
	}
	return f.buffer.Write(p)
}

func (f *s3File) Read(p []byte) (n int, err error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	if f.reader == nil {
		// Lazy load from S3
		result, err := f.fs.client.GetObject(
			context.Background(),
			&s3.GetObjectInput{
				Bucket: aws.String(f.fs.bucketName),
				Key:    aws.String(f.path),
			},
		)
		if err != nil {
			return 0, err
		}
		f.reader = result.Body
	}
	return f.reader.Read(p)
}

func (f *s3File) ReadAt(p []byte, off int64) (n int, err error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	// S3 doesn't support efficient ReadAt, so we read from the offset
	result, err := f.fs.client.GetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(f.fs.bucketName),
			Key:    aws.String(f.path),
			Range:  aws.String(fmt.Sprintf("bytes=%d-%d", off, off+int64(len(p))-1)),
		},
	)
	if err != nil {
		return 0, err
	}
	defer result.Body.Close()
	return io.ReadFull(result.Body, p)
}

func (f *s3File) Seek(offset int64, whence int) (int64, error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	// For simplicity, we don't support seek on S3 files
	return 0, errors.New("seek not supported on S3 files")
}

func (f *s3File) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true

	// If we have a buffer, write it to S3
	if f.buffer != nil && f.buffer.Len() > 0 {
		_, err := f.fs.client.PutObject(
			context.Background(),
			&s3.PutObjectInput{
				Bucket: aws.String(f.fs.bucketName),
				Key:    aws.String(f.path),
				Body:   bytes.NewReader(f.buffer.Bytes()),
			},
		)
		if err != nil {
			return err
		}
	}

	if f.reader != nil {
		return f.reader.Close()
	}
	return nil
}

func (f *s3File) Lock() error {
	return errors.New("lock not supported on S3 files")
}

func (f *s3File) Unlock() error {
	return errors.New("unlock not supported on S3 files")
}

func (f *s3File) Truncate(size int64) error {
	if f.closed {
		return os.ErrClosed
	}
	// For simplicity, we don't support truncate
	return errors.New("truncate not supported on S3 files")
}
