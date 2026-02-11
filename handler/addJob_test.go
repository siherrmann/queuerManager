package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuer/helper"
	"github.com/siherrmann/queuerManager/database"
	qmModel "github.com/siherrmann/queuerManager/model"
	"github.com/siherrmann/queuerManager/upload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddJobViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("AddJobView renders successfully", func(t *testing.T) {
		// Create a test task first
		_, err := tdb.InsertTask(&qmModel.Task{
			Key:  "test-add-job-view",
			Name: "Test Add Job View",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.AddJobView(c)
		require.NoError(t, err)

		// View functions render HTML templates
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("HX-Push-Url"), "/")
		assert.Contains(t, rec.Header().Get("HX-Retarget"), "#body")
	})
}

func TestAddJobConfigViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("AddJobConfigView with valid task", func(t *testing.T) {
		// Create a test task
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:         "test-config-view",
			Name:        "Test Config View",
			Description: "Test task for config view",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/task/"+task.Key, nil)
		// Add CSRF token to request context for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "taskKey", Value: task.Key}})

		err = handler.AddJobConfigView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("HX-Push-Url"), "/task/"+task.Key)
		assert.Contains(t, rec.Header().Get("HX-Retarget"), "#body")
	})

	t.Run("AddJobConfigView with non-existent task", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/task/nonexistent-task", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "taskKey", Value: "nonexistent-task"}})

		err := handler.AddJobConfigView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Missing or non-existent task name")
	})
}
