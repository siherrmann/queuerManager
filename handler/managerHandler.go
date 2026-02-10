package handler

import (
	"net/http"

	"github.com/siherrmann/queuerManager/database"
	"github.com/siherrmann/queuerManager/upload"

	"github.com/labstack/echo/v5"
	"github.com/siherrmann/validator"
)

type ManagerHandler struct {
	filesystem upload.Filesystem
	validator  *validator.Validator
	taskDB     *database.TaskDBHandler
}

func NewManagerHandler(filesystem upload.Filesystem, taskDB *database.TaskDBHandler) *ManagerHandler {
	return &ManagerHandler{
		filesystem: filesystem,
		validator:  validator.NewValidator(),
		taskDB:     taskDB,
	}
}

// Health check handler
func (m *ManagerHandler) HealthCheck(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "queuer-manager",
	})
}
