package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuer/helper"
	"github.com/siherrmann/queuerManager/database"
	"github.com/siherrmann/queuerManager/upload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkerHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	// Get the queuer's own worker RID
	var workerRID uuid.UUID
	t.Run("Setup - Get a worker RID", func(t *testing.T) {
		workerRid := queue.GetCurrentWorkerRID()
		require.NotEqual(t, uuid.Nil, workerRid)
		workerRID = workerRid
	})

	t.Run("GetWorker with valid RID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers/"+workerRID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: workerRID.String()}})

		err := handler.GetWorker(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var worker map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &worker)
		require.NoError(t, err)
		assert.Equal(t, workerRID.String(), worker["rid"])
	})

	t.Run("GetWorker with invalid RID format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers/invalid-uuid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: "invalid-uuid"}})

		err := handler.GetWorker(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid worker RID format")
	})

	t.Run("GetWorker with non-existent RID", func(t *testing.T) {
		nonExistentRID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers/"+nonExistentRID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: nonExistentRID.String()}})

		err := handler.GetWorker(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Worker not found")
	})
}

func TestGetWorkersHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("GetWorkers with default pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetWorkers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var workers []map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &workers)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(workers), 1)
	})

	t.Run("GetWorkers with custom limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers?limit=5", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetWorkers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var workers []map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &workers)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(workers), 5)
	})

	t.Run("GetWorkers with custom lastId and limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers?lastId=0&limit=3", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetWorkers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var workers []map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &workers)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(workers), 3)
	})

	t.Run("GetWorkers with invalid lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers?lastId=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetWorkers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid lastId format")
	})

	t.Run("GetWorkers with negative lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers?lastId=-1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetWorkers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid lastId format")
	})

	t.Run("GetWorkers with invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers?limit=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetWorkers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid limit")
	})

	t.Run("GetWorkers with limit too high", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers?limit=101", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetWorkers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid limit")
	})

	t.Run("GetWorkers with limit zero", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers?limit=0", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetWorkers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid limit")
	})
}

func TestWorkerViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("WorkerView with valid worker RID", func(t *testing.T) {
		// Get a worker from the queue
		workers, err := queue.GetWorkers(0, 1)
		require.NoError(t, err)
		require.Greater(t, len(workers), 0)

		workerRID := workers[0].RID

		req := httptest.NewRequest(http.MethodGet, "/worker?rid="+workerRID.String(), nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.WorkerView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Header().Get("HX-Push-Url"), "/worker?rid=")
	})

	t.Run("WorkerView with missing RID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/worker", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.WorkerView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Missing worker RID")
	})
}

func TestWorkersViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("WorkersView renders successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/workers", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.WorkersView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Header().Get("HX-Push-Url"), "/workers")
	})

	t.Run("WorkersView with search parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/workers?search=test", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.WorkersView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("WorkersView with lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/workers?lastId=1", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.WorkersView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("WorkersView with limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/workers?limit=5", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.WorkersView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("WorkersView with invalid lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/workers?lastId=invalid", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.WorkersView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid lastId")
	})
}

func TestStopWorkersViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("StopWorkersView with valid but not existing RID", func(t *testing.T) {
		rid := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/worker/stopWorkers?rid="+rid.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.StopWorkersView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to stop worker")
	})

	t.Run("StopWorkersView with no RIDs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/worker/stopWorkers", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.StopWorkersView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "No worker RIDs provided")
	})

	t.Run("StopWorkersView with invalid RID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/worker/stopWorkers?rid=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.StopWorkersView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid worker RID")
	})
}

func TestStopWorkersGracefullyViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("StopWorkersGracefullyView with valid RID", func(t *testing.T) {
		rid := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/worker/stopWorkersGracefully?rid="+rid.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.StopWorkersGracefullyView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to gracefully stop worker")
	})

	t.Run("StopWorkersGracefullyView with no RIDs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/worker/stopWorkersGracefully", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.StopWorkersGracefullyView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "No worker RIDs provided")
	})

	t.Run("StopWorkersGracefullyView with invalid RID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/worker/stopWorkersGracefully?rid=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.StopWorkersGracefullyView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid worker RID")
	})
}
