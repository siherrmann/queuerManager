package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuer/helper"
	"github.com/siherrmann/queuerManager/database"
	qmModel "github.com/siherrmann/queuerManager/model"
	"github.com/siherrmann/queuerManager/upload"
	vm "github.com/siherrmann/validator/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTaskHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("AddTask with valid data", func(t *testing.T) {
		taskData := map[string]string{
			"key":         "test-add-task",
			"name":        "Test Add Task",
			"description": "A test task",
			"validations": `[{"Key":"param1","Type":"string","Requirement":"min1"}]`,
		}

		formData := strings.NewReader(fmt.Sprintf(
			"key=%s&name=%s&description=%s&validations=%s",
			taskData["key"], taskData["name"], taskData["description"], taskData["validations"],
		))

		req := httptest.NewRequest(http.MethodPost, "/api/task/addTask", formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.AddTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, rec.Code)

		// Verify task was created
		task, err := tdb.SelectTaskByKey("test-add-task")
		require.NoError(t, err)
		assert.Equal(t, "Test Add Task", task.Name)
		assert.Equal(t, "A test task", task.Description)
		assert.Len(t, task.InputParameters, 1)
	})

	t.Run("AddTask with missing key", func(t *testing.T) {
		formData := strings.NewReader("name=Test&description=Test")

		req := httptest.NewRequest(http.MethodPost, "/api/task/addTask", formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.AddTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Task key is required")
	})

	t.Run("AddTask with missing name", func(t *testing.T) {
		formData := strings.NewReader("key=test-key&description=Test")

		req := httptest.NewRequest(http.MethodPost, "/api/task/addTask", formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.AddTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Task name is required")
	})

	t.Run("AddTask with invalid validations JSON", func(t *testing.T) {
		formData := strings.NewReader("key=test-key&name=Test&validations=invalid-json")

		req := httptest.NewRequest(http.MethodPost, "/api/task/addTask", formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.AddTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid validations JSON")
	})
}

func TestUpdateTaskHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("UpdateTask with valid data", func(t *testing.T) {
		// First create a task
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:  "test-update-task",
			Name: "Original Name",
		})
		require.NoError(t, err)

		formData := strings.NewReader(fmt.Sprintf(
			"key=%s&name=%s&description=%s",
			"test-update-task-renamed", "Updated Name", "Updated Description",
		))

		req := httptest.NewRequest(http.MethodPatch, "/api/task/updateTask?rid="+task.RID.String(), formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.UpdateTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify task was updated
		updatedTask, err := tdb.SelectTask(task.RID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updatedTask.Name)
		assert.Equal(t, "Updated Description", updatedTask.Description)
	})

	t.Run("UpdateTask with invalid RID", func(t *testing.T) {
		formData := strings.NewReader("key=test-key&name=Test")

		req := httptest.NewRequest(http.MethodPatch, "/api/task/updateTask?rid=invalid-uuid", formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UpdateTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid task RID")
	})

	t.Run("UpdateTask with non-existent RID", func(t *testing.T) {
		nonExistentRID := uuid.New()
		formData := strings.NewReader("key=test-key&name=Test")

		req := httptest.NewRequest(http.MethodPatch, "/api/task/updateTask?rid="+nonExistentRID.String(), formData)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UpdateTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to update task")
	})
}

func TestDeleteTasksHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("DeleteTasks with valid RIDs", func(t *testing.T) {
		// Create tasks to delete
		task1, err := tdb.InsertTask(&qmModel.Task{
			Key:  "test-delete-task-1",
			Name: "Delete Task 1",
		})
		require.NoError(t, err)

		task2, err := tdb.InsertTask(&qmModel.Task{
			Key:  "test-delete-task-2",
			Name: "Delete Task 2",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete,
			fmt.Sprintf("/api/task/deleteTasks?rid=%s&rid=%s", task1.RID, task2.RID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.DeleteTasks(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify tasks were deleted
		_, err = tdb.SelectTask(task1.RID)
		assert.Error(t, err)

		_, err = tdb.SelectTask(task2.RID)
		assert.Error(t, err)
	})

	t.Run("DeleteTasks with no RIDs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/task/deleteTasks", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.DeleteTasks(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Missing task RID")
	})
}

func TestGetTaskHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("GetTask with valid RID", func(t *testing.T) {
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:         "test-get-task",
			Name:        "Test Get Task",
			Description: "Description",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/task/getTask/"+task.RID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: task.RID.String()}})

		err = handler.GetTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var fetchedTask qmModel.Task
		err = json.Unmarshal(rec.Body.Bytes(), &fetchedTask)
		require.NoError(t, err)
		assert.Equal(t, task.RID, fetchedTask.RID)
		assert.Equal(t, "Test Get Task", fetchedTask.Name)
	})

	t.Run("GetTask with invalid RID format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/task/getTask/invalid-uuid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: "invalid-uuid"}})

		err := handler.GetTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid task RID format")
	})

	t.Run("GetTask with non-existent RID", func(t *testing.T) {
		nonExistentRID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/task/getTask/"+nonExistentRID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "rid", Value: nonExistentRID.String()}})

		err := handler.GetTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Task not found")
	})
}

func TestGetTaskByNameHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("GetTaskByName with valid name", func(t *testing.T) {
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:  "test-get-by-name",
			Name: "Test Get By Name",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/task/getTaskByName/"+task.Key, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "name", Value: task.Key}})

		err = handler.GetTaskByName(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var fetchedTask qmModel.Task
		err = json.Unmarshal(rec.Body.Bytes(), &fetchedTask)
		require.NoError(t, err)
		assert.Equal(t, task.Key, fetchedTask.Key)
	})

	t.Run("GetTaskByName with non-existent name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/task/getTaskByName/NonExistentTask", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues([]echo.PathValue{{Name: "name", Value: "NonExistentTask"}})

		err := handler.GetTaskByName(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Task not found")
	})
}

func TestGetTasksHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("GetTasks with default pagination", func(t *testing.T) {
		// Create multiple tasks
		for i := 0; i < 5; i++ {
			_, err := tdb.InsertTask(&qmModel.Task{
				Key:  fmt.Sprintf("test-get-tasks-%d", i),
				Name: fmt.Sprintf("Test Task %d", i),
			})
			require.NoError(t, err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/task/getTasks", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetTasks(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var tasks []*qmModel.Task
		err = json.Unmarshal(rec.Body.Bytes(), &tasks)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)
		assert.LessOrEqual(t, len(tasks), 10) // Default limit
	})

	t.Run("GetTasks with custom pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/task/getTasks?lastId=0&limit=3", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetTasks(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var tasks []*qmModel.Task
		err = json.Unmarshal(rec.Body.Bytes(), &tasks)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(tasks), 3)
	})

	t.Run("GetTasks with invalid lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/task/getTasks?lastId=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetTasks(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid lastId")
	})

	t.Run("GetTasks with invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/task/getTasks?limit=200", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetTasks(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid limit")
	})
}

func TestExportTaskHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("ExportTask with valid RIDs", func(t *testing.T) {
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:         "test-export-task",
			Name:        "Test Export Task",
			Description: "Export test",
			InputParameters: []vm.Validation{
				{Key: "param1", Type: "string", Requirement: "min1"},
			},
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/task/exportTask?rid="+task.RID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.ExportTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "tasks_export.json")

		var exportedTasks []map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &exportedTasks)
		require.NoError(t, err)
		assert.Len(t, exportedTasks, 1)
		assert.Equal(t, "test-export-task", exportedTasks[0]["key"])
	})

	t.Run("ExportTask with no RIDs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/task/exportTask", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.ExportTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Missing task RIDs")
	})
}

func TestImportTaskHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("ImportTask with valid file", func(t *testing.T) {
		tasksJSON := `[
			{
				"key": "test-import-task",
				"name": "Test Import Task",
				"description": "Import test",
				"input_parameters": [{"Key": "param1", "Type": "string", "Requirement": "min1"}],
				"input_parameters_keyed": [],
				"output_parameters": []
			}
		]`

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("task_file", "tasks.json")
		require.NoError(t, err)
		_, err = part.Write([]byte(tasksJSON))
		require.NoError(t, err)
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/task/importTask", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.ImportTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "Successfully imported 1 tasks")

		// Verify task was imported
		task, err := tdb.SelectTaskByKey("test-import-task")
		require.NoError(t, err)
		assert.Equal(t, "Test Import Task", task.Name)
	})

	t.Run("ImportTask with no file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/task/importTask", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.ImportTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "No file uploaded")
	})

	t.Run("ImportTask with invalid JSON", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("task_file", "tasks.json")
		require.NoError(t, err)
		_, err = part.Write([]byte("invalid json"))
		require.NoError(t, err)
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/task/importTask", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.ImportTask(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid JSON format")
	})
}

func TestTaskViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("TaskView with valid RID", func(t *testing.T) {
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:  "test-task-view",
			Name: "Test Task View",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/task?rid="+task.RID.String(), nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.TaskView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Header().Get("HX-Push-Url"), "/task?rid=")
	})

	t.Run("TaskView with missing RID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/task", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.TaskView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Missing task RID")
	})
}

func TestTasksViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("TasksView renders successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.TasksView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Header().Get("HX-Push-Url"), "/tasks")
	})

	t.Run("TasksView with search parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks?search=test", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.TasksView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("TasksView with lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks?lastId=1", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.TasksView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("TasksView with limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks?limit=5", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.TasksView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})

	t.Run("TasksView with invalid lastId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks?lastId=invalid", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.TasksView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid lastId")
	})
}

func TestAddTaskPopupViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("AddTaskPopupView renders successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/task/addTaskPopup", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.AddTaskPopupView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})
}

func TestUpdateTaskPopupViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("UpdateTaskPopupView renders successfully", func(t *testing.T) {
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:  "test-update-popup",
			Name: "Test Update Popup",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/task/updateTaskPopup?rid="+task.RID.String(), nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.UpdateTaskPopupView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})
}

func TestDeleteTaskPopupViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("DeleteTaskPopupView renders successfully", func(t *testing.T) {
		task, err := tdb.InsertTask(&qmModel.Task{
			Key:  "test-delete-popup",
			Name: "Test Delete Popup",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/task/deleteTaskPopup?rid="+task.RID.String(), nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler.DeleteTaskPopupView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})
}

func TestImportTaskPopupViewHandler(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	t.Run("ImportTaskPopupView renders successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/task/importTaskPopup", nil)
		// Add CSRF token for templ rendering
		ctx := context.WithValue(req.Context(), "gorilla.csrf.Token", "test-csrf-token")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.ImportTaskPopupView(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=UTF-8", rec.Header().Get("Content-Type"))
	})
}
