package handler

import (
	"bytes"
	"context"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuer/helper"
	"github.com/siherrmann/queuerManager/database"
	"github.com/siherrmann/queuerManager/upload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadFilesHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("UploadFiles with single file", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("files", "test.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("test content"))
		require.NoError(t, err)
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/file/uploadFiles", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.UploadFiles(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "1 file(s) uploaded successfully")

		// Verify file was uploaded
		files, err := fs.ListFiles()
		require.NoError(t, err)
		found := false
		for _, file := range files {
			if file.Name == "test.txt" {
				found = true
				break
			}
		}
		assert.True(t, found, "File should be in the filesystem")
	})

	t.Run("UploadFiles with multiple files", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add first file
		part1, err := writer.CreateFormFile("files", "file1.txt")
		require.NoError(t, err)
		_, err = part1.Write([]byte("content 1"))
		require.NoError(t, err)

		// Add second file
		part2, err := writer.CreateFormFile("files", "file2.txt")
		require.NoError(t, err)
		_, err = part2.Write([]byte("content 2"))
		require.NoError(t, err)

		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/file/uploadFiles", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.UploadFiles(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "2 file(s) uploaded successfully")

		// Verify both files were uploaded
		files, err := fs.ListFiles()
		require.NoError(t, err)
		file1Found := false
		file2Found := false
		for _, file := range files {
			if file.Name == "file1.txt" {
				file1Found = true
			}
			if file.Name == "file2.txt" {
				file2Found = true
			}
		}
		assert.True(t, file1Found, "File 1 should be in the filesystem")
		assert.True(t, file2Found, "File 2 should be in the filesystem")
	})

	t.Run("UploadFiles with no files", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/file/uploadFiles", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UploadFiles(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "No files found in the request")
	})
}

func TestDeleteFileHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("DeleteFile with existing file", func(t *testing.T) {
		// First upload a file
		err := fs.Write("delete-test.txt", strings.NewReader("content"), 7)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, "/api/file/deleteFile/delete-test.txt", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "filename", Value: "delete-test.txt"}})

		err = handler.DeleteFile(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "deleted successfully")

		// Verify file was deleted
		files, err := fs.ListFiles()
		require.NoError(t, err)
		for _, file := range files {
			assert.NotEqual(t, "delete-test.txt", file.Name)
		}
	})

	t.Run("DeleteFile with non-existent file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/file/deleteFile/nonexistent.txt", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "filename", Value: "nonexistent.txt"}})

		err := handler.DeleteFile(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to delete file")
	})
}

func TestDeleteFilesHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("DeleteFiles with multiple existing files", func(t *testing.T) {
		// Upload test files
		err := fs.Write("file-to-delete-1.txt", strings.NewReader("content 1"), 9)
		require.NoError(t, err)
		err = fs.Write("file-to-delete-2.txt", strings.NewReader("content 2"), 9)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete,
			"/api/file/deleteFiles?name=file-to-delete-1.txt&name=file-to-delete-2.txt", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.DeleteFiles(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "2 file(s) deleted successfully")

		// Verify files were deleted
		files, err := fs.ListFiles()
		require.NoError(t, err)
		for _, file := range files {
			assert.NotEqual(t, "file-to-delete-1.txt", file.Name)
			assert.NotEqual(t, "file-to-delete-2.txt", file.Name)
		}
	})

	t.Run("DeleteFiles with no file names", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/file/deleteFiles", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.DeleteFiles(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "No file names provided")
	})

	t.Run("DeleteFiles with mixed existing and non-existent files", func(t *testing.T) {
		// Upload one file
		err := fs.Write("existing-file.txt", strings.NewReader("content"), 7)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete,
			"/api/file/deleteFiles?name=existing-file.txt&name=nonexistent-file.txt", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.DeleteFiles(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Deleted 1 file(s)")
		assert.Contains(t, rec.Body.String(), "but 1 failed")
	})
}

func TestFileViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("FileView with missing filename", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/file", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.FileView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "File name is required")
	})

	t.Run("FileView with non-existent file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/file?name=nonexistent.txt", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.FileView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "File not found")
	})
}

func TestFilesViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("FilesView basic listing", func(t *testing.T) {
		// Upload some test files
		err := fs.Write("list-test-1.txt", strings.NewReader("content"), 7)
		require.NoError(t, err)
		err = fs.Write("list-test-2.txt", strings.NewReader("content"), 7)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.FilesView(c)
		require.NoError(t, err)

		// View handlers return HTML, so just check status
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("FilesView with search filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files?search=list-test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.FilesView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestDeleteFilePopupViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("DeleteFilePopupView with no file names", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/file/deleteFilePopup", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.DeleteFilePopupView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "No file names provided")
	})

	t.Run("DeleteFilePopupView with file names", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/file/deleteFilePopup?name=file1.txt&name=file2.txt", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.DeleteFilePopupView(c)
		require.NoError(t, err)

		// Should render popup successfully
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestAddFilePopupViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("AddFilePopupView renders successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/file/addFilePopup", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.AddFilePopupView(c)
		require.NoError(t, err)

		// View handlers return HTML templates
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
