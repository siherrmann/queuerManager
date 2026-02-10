package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/siherrmann/queuerManager/model"
	"github.com/siherrmann/queuerManager/view/screens"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	vm "github.com/siherrmann/validator/model"
)

// =======API Handlers=======

// AddTask handles the addition of a new task
func (m *ManagerHandler) AddTask(c *echo.Context) error {
	var requestData struct {
		Key              string `json:"key" form:"key"`
		Name             string `json:"name" form:"name"`
		Description      string `json:"description" form:"description"`
		Validations      string `json:"validations" form:"validations"`
		ValidationsKeyed string `json:"validations_keyed" form:"validations_keyed"`
		OutputParameters string `json:"output_parameters" form:"output_parameters"`
	}

	if err := c.Bind(&requestData); err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
	}

	if requestData.Key == "" {
		return renderPopupOrJson(c, http.StatusBadRequest, "Task key is required")
	}

	if requestData.Name == "" {
		return renderPopupOrJson(c, http.StatusBadRequest, "Task name is required")
	}

	// Parse validations JSON
	var validations []vm.Validation
	if requestData.Validations != "" {
		if err := json.Unmarshal([]byte(requestData.Validations), &validations); err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid validations JSON: %v", err))
		}
	}

	// Parse validations_keyed JSON
	var validationsKeyed []vm.Validation
	if requestData.ValidationsKeyed != "" {
		if err := json.Unmarshal([]byte(requestData.ValidationsKeyed), &validationsKeyed); err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid validations_keyed JSON: %v", err))
		}
	}

	// Parse output_parameters JSON
	var outputParameters []vm.Validation
	if requestData.OutputParameters != "" {
		if err := json.Unmarshal([]byte(requestData.OutputParameters), &outputParameters); err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid output_parameters JSON: %v", err))
		}
	}

	task := &model.Task{
		Key:                  requestData.Key,
		Name:                 requestData.Name,
		Description:          requestData.Description,
		InputParameters:      validations,
		InputParametersKeyed: validationsKeyed,
		OutputParameters:     outputParameters,
	}

	insertedTask, err := m.taskDB.InsertTask(task)
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to add task: %v", err))
	}

	c.Response().Header().Add("HX-Redirect", "/tasks")

	return renderPopupOrJson(c, http.StatusCreated, "Task added successfully", insertedTask)
}

// UpdateTask handles updating an existing task
func (m *ManagerHandler) UpdateTask(c *echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if len(ridStrings) == 0 || !ok {
		return renderPopupOrJson(c, http.StatusBadRequest, "Missing task RID")
	}

	rid, err := uuid.Parse(ridStrings[0])
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid task RID: %v", err))
	}

	var requestData struct {
		Key              string `json:"key" form:"key"`
		Name             string `json:"name" form:"name"`
		Description      string `json:"description" form:"description"`
		Validations      string `json:"validations" form:"validations"`
		ValidationsKeyed string `json:"validations_keyed" form:"validations_keyed"`
		OutputParameters string `json:"output_parameters" form:"output_parameters"`
	}

	if err := c.Bind(&requestData); err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
	}

	if requestData.Key == "" {
		return renderPopupOrJson(c, http.StatusBadRequest, "Task key is required")
	}

	if requestData.Name == "" {
		return renderPopupOrJson(c, http.StatusBadRequest, "Task name is required")
	}

	// Parse validations JSON
	var validations []vm.Validation
	if requestData.Validations != "" {
		if err := json.Unmarshal([]byte(requestData.Validations), &validations); err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid validations JSON: %v", err))
		}
	}

	// Parse validations_keyed JSON
	var validationsKeyed []vm.Validation
	if requestData.ValidationsKeyed != "" {
		if err := json.Unmarshal([]byte(requestData.ValidationsKeyed), &validationsKeyed); err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid validations_keyed JSON: %v", err))
		}
	}

	// Parse output_parameters JSON
	var outputParameters []vm.Validation
	if requestData.OutputParameters != "" {
		if err := json.Unmarshal([]byte(requestData.OutputParameters), &outputParameters); err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid output_parameters JSON: %v", err))
		}
	}

	task := &model.Task{
		RID:                  rid,
		Key:                  requestData.Key,
		Name:                 requestData.Name,
		Description:          requestData.Description,
		InputParameters:      validations,
		InputParametersKeyed: validationsKeyed,
		OutputParameters:     outputParameters,
	}

	updatedTask, err := m.taskDB.UpdateTask(task)
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to update task: %v", err))
	}

	c.Response().Header().Add("HX-Redirect", "/tasks")

	return renderPopupOrJson(c, http.StatusOK, "Task updated successfully", updatedTask)
}

// DeleteTasks deletes multiple tasks by RIDs
func (m *ManagerHandler) DeleteTasks(c *echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if len(ridStrings) == 0 || !ok {
		return renderPopupOrJson(c, http.StatusBadRequest, "Missing task RID")
	}

	// Delete each task
	deletedCount := 0
	var errors []string
	for _, ridStr := range ridStrings {
		rid, err := uuid.Parse(ridStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Invalid RID %s: %v", ridStr, err))
			continue
		}

		err = m.taskDB.DeleteTask(rid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete task %s: %v", ridStr, err))
			continue
		}
		deletedCount++
	}

	// Trigger table refresh
	c.Response().Header().Add("HX-Trigger", "getTasks")

	if len(errors) > 0 {
		return renderPopupOrJson(c, http.StatusPartialContent, fmt.Sprintf("Deleted %d tasks. Errors: %v", deletedCount, errors))
	}

	return renderPopupOrJson(c, http.StatusOK, fmt.Sprintf("Successfully deleted %d task(s)", deletedCount))
}

// GetTask retrieves a specific task by RID
func (m *ManagerHandler) GetTask(c *echo.Context) error {
	ridStr := c.Param("rid")
	rid, err := uuid.Parse(ridStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid task RID format")
	}

	task, err := m.taskDB.SelectTask(rid)
	if err != nil {
		return c.String(http.StatusNotFound, "Task not found")
	}

	return c.JSON(http.StatusOK, task)
}

// GetTaskByName retrieves a specific task by name
func (m *ManagerHandler) GetTaskByName(c *echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Task name is required")
	}

	task, err := m.taskDB.SelectTaskByKey(name)
	if err != nil {
		return c.String(http.StatusNotFound, "Task not found")
	}

	return c.JSON(http.StatusOK, task)
}

// GetTasks retrieves a paginated list of tasks
func (m *ManagerHandler) GetTasks(c *echo.Context) error {
	lastIdStr := c.QueryParam("lastId")
	limitStr := c.QueryParam("limit")

	// Parse lastId with default
	lastId := 0
	if lastIdStr != "" {
		parsedLastId, err := strconv.Atoi(lastIdStr)
		if err != nil || parsedLastId < 0 {
			return c.String(http.StatusBadRequest, "Invalid lastId format")
		}
		lastId = parsedLastId
	}

	// Parse limit with default
	limit := 10
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			return c.String(http.StatusBadRequest, "Invalid limit (must be 1-100)")
		}
		limit = parsedLimit
	}

	tasks, err := m.taskDB.SelectAllTasks(lastId, limit)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve tasks")
	}

	return c.JSON(http.StatusOK, tasks)
}

// =======View Handlers=======

// TaskView renders the task detail view
func (m *ManagerHandler) TaskView(c *echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if len(ridStrings) == 0 || !ok {
		return renderPopupOrJson(c, http.StatusBadRequest, "Missing task RID")
	}

	rid, err := uuid.Parse(ridStrings[0])
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid task RID: %v", err))
	}

	task, err := m.taskDB.SelectTask(rid)
	if err != nil {
		return renderPopupOrJson(c, http.StatusNotFound, "Task not found")
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/task?rid=%v", rid))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.Task(task))
}

// TasksView renders the tasks list view
func (m *ManagerHandler) TasksView(c *echo.Context) error {
	lastIdStr := c.QueryParam("lastId")
	limitStr := c.QueryParam("limit")
	search := c.QueryParam("search")

	// Parse lastId with default
	lastId := 0
	if lastIdStr != "" {
		parsedLastId, err := strconv.Atoi(lastIdStr)
		if err != nil || parsedLastId < 0 {
			return c.String(http.StatusBadRequest, "Invalid lastId format")
		}
		lastId = parsedLastId
	}

	// Parse limit with default
	limit := 100
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			return c.String(http.StatusBadRequest, "Invalid limit (must be 1-100)")
		}
		limit = parsedLimit
	}

	var tasks []*model.Task
	var err error
	if search != "" {
		tasks, err = m.taskDB.SelectAllTasksBySearch(search, lastId, limit)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to search tasks")
		}
	} else {
		tasks, err = m.taskDB.SelectAllTasks(lastId, limit)
		if err != nil {
			log.Printf("Error retrieving tasks: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to retrieve tasks")
		}
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/tasks?search=%s&limit=%d&lastId=%d", search, limit, lastId))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.Tasks(tasks, search))
}

// =======Popup Handlers=======

// AddTaskPopupView renders the add task popup
func (m *ManagerHandler) AddTaskPopupView(c *echo.Context) error {
	return renderPopup(c, screens.AddTaskPopup())
}

// UpdateTaskPopupView renders the update task popup
func (m *ManagerHandler) UpdateTaskPopupView(c *echo.Context) error {
	ridStr := c.QueryParam("rid")
	rid, err := uuid.Parse(ridStr)
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid task RID: %v", err))
	}

	task, err := m.taskDB.SelectTask(rid)
	if err != nil {
		return renderPopupOrJson(c, http.StatusNotFound, "Task not found")
	}

	return renderPopup(c, screens.UpdateTaskPopup(task))
}

// DeleteTaskPopupView renders the delete task confirmation popup
func (m *ManagerHandler) DeleteTaskPopupView(c *echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if len(ridStrings) == 0 || !ok {
		return renderPopupOrJson(c, http.StatusBadRequest, "Missing task RIDs")
	}

	log.Printf("rids: %v", ridStrings[0])

	return renderPopup(c, screens.DeleteTaskPopup(ridStrings))
}

// ImportTaskPopupView renders the import task popup
func (m *ManagerHandler) ImportTaskPopupView(c *echo.Context) error {
	return renderPopup(c, screens.ImportTaskPopup())
}

// ExportTask exports selected tasks as JSON array file
func (m *ManagerHandler) ExportTask(c *echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if len(ridStrings) == 0 || !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing task RIDs"})
	}

	var exportTasks []map[string]interface{}

	for _, ridStr := range ridStrings {
		rid, err := uuid.Parse(ridStr)
		if err != nil {
			log.Printf("Invalid task RID: %s, skipping", ridStr)
			continue
		}

		task, err := m.taskDB.SelectTask(rid)
		if err != nil {
			log.Printf("Task not found: %s, skipping", ridStr)
			continue
		}

		// Create a clean export without ID and timestamps
		exportTask := map[string]interface{}{
			"key":                    task.Key,
			"name":                   task.Name,
			"description":            task.Description,
			"input_parameters":       task.InputParameters,
			"input_parameters_keyed": task.InputParametersKeyed,
			"output_parameters":      task.OutputParameters,
		}
		exportTasks = append(exportTasks, exportTask)
	}

	if len(exportTasks) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "No valid tasks found to export"})
	}

	jsonData, err := json.MarshalIndent(exportTasks, "", "  ")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to marshal tasks"})
	}

	filename := "tasks_export.json"
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Response().Header().Set("Content-Type", "application/json")

	return c.Blob(http.StatusOK, "application/json", jsonData)
}

// ImportTask imports tasks from JSON array file
func (m *ManagerHandler) ImportTask(c *echo.Context) error {
	file, err := c.FormFile("task_file")
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, "No file uploaded")
	}

	src, err := file.Open()
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, "Failed to open file")
	}
	defer src.Close()

	var tasksData []struct {
		Key                  string          `json:"key"`
		Name                 string          `json:"name"`
		Description          string          `json:"description"`
		InputParameters      []vm.Validation `json:"input_parameters"`
		InputParametersKeyed []vm.Validation `json:"input_parameters_keyed"`
		OutputParameters     []vm.Validation `json:"output_parameters"`
	}

	if err := json.NewDecoder(src).Decode(&tasksData); err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid JSON format: %v", err))
	}

	if len(tasksData) == 0 {
		return renderPopupOrJson(c, http.StatusBadRequest, "No tasks found in JSON file")
	}

	importedCount := 0
	var errors []string

	for _, taskData := range tasksData {
		if taskData.Key == "" {
			errors = append(errors, "Skipped task with empty key")
			continue
		}

		task := &model.Task{
			Key:                  taskData.Key,
			Name:                 taskData.Name,
			Description:          taskData.Description,
			InputParameters:      taskData.InputParameters,
			InputParametersKeyed: taskData.InputParametersKeyed,
			OutputParameters:     taskData.OutputParameters,
		}

		_, err := m.taskDB.InsertTask(task)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to import task '%s': %v", taskData.Key, err))
			continue
		}
		importedCount++
	}

	c.Response().Header().Add("HX-Redirect", "/tasks")

	if len(errors) > 0 {
		errorMsg := fmt.Sprintf("Imported %d tasks with errors: %v", importedCount, errors)
		return renderPopupOrJson(c, http.StatusPartialContent, errorMsg)
	}

	return renderPopupOrJson(c, http.StatusCreated, fmt.Sprintf("Successfully imported %d tasks", importedCount))
}
