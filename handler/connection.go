package handler

import (
	"manager/helper"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetConnections retrieves all active connections
func (m *ManagerHandler) GetConnections(c echo.Context) error {
	connections, err := helper.Queuer.GetConnections()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve connections")
	}

	return c.JSON(http.StatusOK, connections)
}
