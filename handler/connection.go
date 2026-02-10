package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// GetConnections retrieves all active connections
func (m *ManagerHandler) GetConnections(c *echo.Context) error {
	connections, err := m.Queuer.GetConnections()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve connections")
	}

	return c.JSON(http.StatusOK, connections)
}
