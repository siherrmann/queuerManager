package handler

import (
	"fmt"
	"net/http"

	"github.com/siherrmann/queuerManager/view/screens"

	"github.com/labstack/echo/v5"
)

func (m *ManagerHandler) AddJobView(c *echo.Context) error {
	tasks, err := m.taskDB.SelectAllTasks(0, 100)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve tasks")
	}

	c.Response().Header().Add("HX-Push-Url", "/")
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.AddJob(tasks))
}

// AddJobConfigView renders a task-specific screen with parameter inputs
func (m *ManagerHandler) AddJobConfigView(c *echo.Context) error {
	taskKey := c.Param("taskKey")
	task, err := m.taskDB.SelectTaskByKey(taskKey)
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, "Missing or non-existent task name")
	}

	files, err := m.filesystem.ListFiles()
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Error listing files: %v", err))
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/task/%s", task.Key))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.AddJobConfig(task, files))
}
