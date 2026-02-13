package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuer/helper"
	"github.com/siherrmann/queuer/model"
	"github.com/siherrmann/queuerManager/database"
	qmModel "github.com/siherrmann/queuerManager/model"
	"github.com/siherrmann/queuerManager/upload"
	vm "github.com/siherrmann/validator/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddJobHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("AddJob with valid task", func(t *testing.T) {
		// First create a task
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:                  "test-job-task",
			Name:                 "Test Job Task",
			Description:          "",
			InputParameters:      []vm.Validation{},
			InputParametersKeyed: []vm.Validation{},
		})
		require.NoError(t, err)

		// Record jobs before
		jobsBefore, _ := queue.GetJobs(0, 100)
		beforeCount := len(jobsBefore)

		req := httptest.NewRequest(http.MethodPost, "/api/job/addJob/"+task.Key, strings.NewReader("{}"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		// Add CSRF token to request context
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "taskKey", Value: task.Key}})

		// handler.AddJob will try to render HTML which requires context, so we expect it might not fully succeed
		// but it should create the job first
		err = handler.AddJob(c)
		require.NoError(t, err)

		// Verify job was actually created regardless of render issues
		time.Sleep(50 * time.Millisecond)
		jobsAfter, _ := queue.GetJobs(0, 100)
		assert.Equal(t, beforeCount+1, len(jobsAfter), "Job should have been created")

		// Find the newly created job
		var newJob *model.Job
		for _, job := range jobsAfter {
			found := true
			for _, beforeJob := range jobsBefore {
				if job.RID == beforeJob.RID {
					found = false
					break
				}
			}
			if found {
				newJob = job
				break
			}
		}
		require.NotNil(t, newJob, "Should find the newly created job")
		assert.Equal(t, task.Key, newJob.TaskName)
	})

	t.Run("AddJob with non-existent task", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/job/addJob/NonExistentTask", strings.NewReader("{}"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "taskKey", Value: "NonExistentTask"}})

		err := handler.AddJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Task not found")
	})
}

func TestGetJobHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("GetJob with valid RID", func(t *testing.T) {
		job, err := queue.AddJob("test-task", nil, 1)
		require.NoError(t, err)

		// Wait a bit for job to be queued
		time.Sleep(100 * time.Millisecond)

		req := httptest.NewRequest(http.MethodGet, "/job/"+job.RID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: job.RID.String()}})

		err = handler.GetJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var fetchedJob model.Job
		err = json.Unmarshal(rec.Body.Bytes(), &fetchedJob)
		require.NoError(t, err)
		assert.Equal(t, job.RID, fetchedJob.RID)
		assert.Equal(t, "test-task", fetchedJob.TaskName)
	})

	t.Run("GetJob with invalid RID format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job/invalid-uuid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: "invalid-uuid"}})

		err := handler.GetJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid job RID format")
	})

	t.Run("GetJob with non-existent RID", func(t *testing.T) {
		nonExistentRID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/job/"+nonExistentRID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: nonExistentRID.String()}})

		err := handler.GetJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Job not found")
	})
}

func TestGetJobsHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("GetJobs with default pagination", func(t *testing.T) {
		// Create multiple jobs
		for i := 0; i < 5; i++ {
			_, err := queue.AddJob("test-task", nil, 1)
			require.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond)

		req := httptest.NewRequest(http.MethodGet, "/api/job/getJobs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetJobs(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var jobs []*model.Job
		err = json.Unmarshal(rec.Body.Bytes(), &jobs)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 1)
		assert.LessOrEqual(t, len(jobs), 10) // Default limit
	})

	t.Run("GetJobs with custom pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/job/getJobs?lastId=0&limit=3", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetJobs(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var jobs []*model.Job
		err = json.Unmarshal(rec.Body.Bytes(), &jobs)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(jobs), 3)
	})

	t.Run("GetJobs with invalid lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/job/getJobs?lastId=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetJobs(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid lastId")
	})

	t.Run("GetJobs with invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/job/getJobs?limit=200", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetJobs(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid limit")
	})
}

func TestCancelJobHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("CancelJob with valid RID", func(t *testing.T) {
		// Create a job
		job, err := queue.AddJob("test-task", nil, 10) // Long running
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		req := httptest.NewRequest(http.MethodPost, "/api/job/cancelJob/"+job.RID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: job.RID.String()}})

		err = handler.CancelJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var cancelledJob model.Job
		err = json.Unmarshal(rec.Body.Bytes(), &cancelledJob)
		require.NoError(t, err)
		assert.Equal(t, job.RID, cancelledJob.RID)
	})

	t.Run("CancelJob with invalid RID format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/job/cancelJob/invalid-uuid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: "invalid-uuid"}})

		err := handler.CancelJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid job RID format")
	})

	t.Run("CancelJob with non-existent RID", func(t *testing.T) {
		nonExistentRID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/job/cancelJob/"+nonExistentRID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: nonExistentRID.String()}})

		err := handler.CancelJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to cancel job")
	})
}

func TestCancelJobsHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("CancelJobs with valid RIDs", func(t *testing.T) {
		// Create multiple jobs
		job1, err := queue.AddJob("test-task", nil, 10)
		require.NoError(t, err)
		job2, err := queue.AddJob("test-task", nil, 10)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		formData := strings.NewReader(fmt.Sprintf("rid=%s&rid=%s", job1.RID.String(), job2.RID.String()))
		req := httptest.NewRequest(http.MethodPost, "/api/job/cancelJobs", formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.CancelJobs(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "2 jobs cancelled successfully")
	})

	t.Run("CancelJobs with no RIDs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/job/cancelJobs", strings.NewReader(""))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.CancelJobs(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to parse form with job RIDs")
	})

	t.Run("CancelJobs with invalid RID format", func(t *testing.T) {
		formData := strings.NewReader("rid=invalid-uuid")
		req := httptest.NewRequest(http.MethodPost, "/api/job/cancelJobs", formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.CancelJobs(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid job RID format")
	})
}

func TestDeleteJobHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("DeleteJob with valid RID", func(t *testing.T) {
		// Create a job
		job, err := queue.AddJob("test-task", nil, 1)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		req := httptest.NewRequest(http.MethodDelete, "/api/job/deleteJob/"+job.RID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: job.RID.String()}})

		err = handler.DeleteJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Job deleted successfully")
	})

	t.Run("DeleteJob with invalid RID format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/job/deleteJob/invalid-uuid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: "invalid-uuid"}})

		err := handler.DeleteJob(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid rid")
	})
}

func TestJobViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("JobView with valid job RID", func(t *testing.T) {
		// Create a job
		job, err := queue.AddJob("test-task", nil, 1)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		req := httptest.NewRequest(http.MethodGet, "/job?rid="+job.RID.String(), nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.JobView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Header().Get("HX-Push-Url"), "/job?rid="+job.RID.String())
	})

	t.Run("JobView with missing RID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.JobView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Missing job RID")
	})

	t.Run("JobView with invalid RID format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job?rid=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.JobView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid job RID format")
	})
}

func TestJobsViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("JobsView renders successfully", func(t *testing.T) {
		// Create some jobs
		for i := 0; i < 3; i++ {
			_, err := queue.AddJob("test-task", nil, 1)
			require.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond)

		req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.JobsView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Header().Get("HX-Push-Url"), "/jobs")
	})

	t.Run("JobsView with search parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jobs?search=test", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.JobsView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("JobsView with lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jobs?lastId=1", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.JobsView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("JobsView with limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jobs?limit=5", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.JobsView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("JobsView with invalid lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jobs?lastId=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.JobsView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid lastId format")
	})

	t.Run("JobsView with invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jobs?limit=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.JobsView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid limit")
	})
}
