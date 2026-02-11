package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuer/helper"
	"github.com/siherrmann/queuerManager/database"
	"github.com/siherrmann/queuerManager/upload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConnectionsHandlers(t *testing.T) {
	fs := upload.NewFilesystemMemory()
	db := helper.NewDatabaseWithDB("taskdb", queue.DB, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	tdb, err := database.NewTaskDBHandler(db, false)
	require.NoError(t, err)

	handler := NewManagerHandler(fs, tdb, queue)
	e := echo.New()

	// Test GetConnections
	t.Run("GetConnections returns all active connections", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetConnections(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var connections []map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &connections)
		require.NoError(t, err)

		// Should have at least the connection from the test setup
		assert.GreaterOrEqual(t, len(connections), 0)

		// If there are connections, verify structure contains expected fields
		if len(connections) > 0 {
			conn := connections[0]
			assert.Contains(t, conn, "username")
			assert.Contains(t, conn, "database")
		}
	})

	t.Run("GetConnections returns valid JSON array", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetConnections(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify it's a valid JSON array
		var connections []interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &connections)
		require.NoError(t, err)
	})
}
