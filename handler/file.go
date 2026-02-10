package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/siherrmann/queuerManager/upload"
	"github.com/siherrmann/queuerManager/view/screens"

	"github.com/labstack/echo/v5"
)

func (m *ManagerHandler) UploadFiles(c *echo.Context) error {
	// Parse multipart form with 32MB max memory
	err := c.Request().ParseMultipartForm(32 << 20)
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Failed to parse multipart form: %v", err))
	}

	form := c.Request().MultipartForm
	defer form.RemoveAll() // Clean up temporary files

	files := form.File["files"]
	if len(files) == 0 {
		return renderPopupOrJson(c, http.StatusBadRequest, "No files found in the request")
	}

	var uploadedFiles []string
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to open file %s: %v", fileHeader.Filename, err))
		}
		defer file.Close()

		// Generate safe filename (you might want to add UUID or timestamp for uniqueness)
		filename := filepath.Base(fileHeader.Filename)
		err = m.filesystem.Write(filename, file, fileHeader.Size)
		if err != nil {
			return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to save file %s: %v", filename, err))
		}

		uploadedFiles = append(uploadedFiles, filename)
	}

	c.Response().Header().Add("HX-Trigger-After-Settle", "reloadFiles")

	return renderPopupOrJson(c, http.StatusOK, fmt.Sprintf("%v file(s) uploaded successfully", len(uploadedFiles)))
}

func (m *ManagerHandler) DeleteFile(c *echo.Context) error {
	filename := c.Param("filename")
	err := m.filesystem.Delete(filename)
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to delete file %s: %v", filename, err))
	}

	c.Response().Header().Add("HX-Trigger-After-Settle", "reloadFiles")

	return renderPopupOrJson(c, http.StatusOK, fmt.Sprintf("File %s deleted successfully", filename))
}

// DeleteFiles deletes multiple files
func (m *ManagerHandler) DeleteFiles(c *echo.Context) error {
	names := c.QueryParams()["name"]
	if len(names) == 0 {
		return renderPopupOrJson(c, http.StatusBadRequest, "No file names provided")
	}

	var deletedFiles []string
	var errors []string

	for _, name := range names {
		err := m.filesystem.Delete(name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
		} else {
			deletedFiles = append(deletedFiles, name)
		}
	}

	c.Response().Header().Add("HX-Trigger-After-Settle", "getFiles")

	if len(errors) > 0 {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Deleted %d file(s), but %d failed: %v", len(deletedFiles), len(errors), errors))
	}

	return renderPopupOrJson(c, http.StatusOK, fmt.Sprintf("%d file(s) deleted successfully", len(deletedFiles)))
}

// FileView renders the file detail view
func (m *ManagerHandler) FileView(c *echo.Context) error {
	filename := c.QueryParam("name")
	if filename == "" {
		return renderPopupOrJson(c, http.StatusBadRequest, "File name is required")
	}

	files, err := m.filesystem.ListFiles()
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list files: %v", err))
	}

	var foundFile *upload.File
	for _, file := range files {
		if file.Name == filename {
			foundFile = &file
			break
		}
	}

	if foundFile == nil {
		return renderPopupOrJson(c, http.StatusNotFound, "File not found")
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/file?name=%s", filename))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.File(*foundFile))
}

// FilesView renders the files list view
func (m *ManagerHandler) FilesView(c *echo.Context) error {
	search := c.QueryParam("search")

	files, err := m.filesystem.ListFiles()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to list files: %v", err),
		})
	}

	if search != "" {
		var filteredFiles []upload.File
		for _, file := range files {
			if strings.Contains(file.Name, search) {
				filteredFiles = append(filteredFiles, file)
			}
		}
		files = filteredFiles
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/files?search=%s", search))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.Files(files, search))
}

// AddFilePopupView renders the add file popup
func (m *ManagerHandler) AddFilePopupView(c *echo.Context) error {
	return renderPopup(c, screens.AddFilePopup())
}

// DeleteFilePopupView renders the delete file popup
func (m *ManagerHandler) DeleteFilePopupView(c *echo.Context) error {
	names := c.QueryParams()["name"]
	if len(names) == 0 {
		return renderPopupOrJson(c, http.StatusBadRequest, "No file names provided")
	}

	return renderPopup(c, screens.DeleteFilePopup(names))
}
