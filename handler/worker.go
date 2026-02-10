package handler

import (
	"fmt"
	"manager/helper"
	"manager/view/screens"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/siherrmann/queuer/model"
)

// GetWorker retrieves a specific worker by RID
func (m *ManagerHandler) GetWorker(c echo.Context) error {
	ridStr := c.Param("rid")
	rid, err := uuid.Parse(ridStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid worker RID format")
	}

	worker, err := helper.Queuer.GetWorker(rid)
	if err != nil {
		return c.String(http.StatusNotFound, "Worker not found")
	}

	return c.JSON(http.StatusOK, worker)
}

// GetWorkers retrieves a paginated list of workers
func (m *ManagerHandler) GetWorkers(c echo.Context) error {
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
	limit := 100
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			return c.String(http.StatusBadRequest, "Invalid limit (must be 1-100)")
		}
		limit = parsedLimit
	}

	workers, err := helper.Queuer.GetWorkers(lastId, limit)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve workers")
	}

	return c.JSON(http.StatusOK, workers)
}

// =======View Handlers=======

// WorkerView renders the worker detail page
func (m *ManagerHandler) WorkerView(c echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if len(ridStrings) == 0 || !ok {
		return renderPopupOrJson(c, http.StatusBadRequest, "Missing worker RID")
	}

	rid, err := uuid.Parse(ridStrings[0])
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid worker RID: %v", err))
	}

	worker, err := helper.Queuer.GetWorker(rid)
	if err != nil {
		return renderPopupOrJson(c, http.StatusNotFound, "Worker not found")
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/worker?rid=%s", rid))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.Worker(worker))
}

// WorkersView renders the workers list page
func (m *ManagerHandler) WorkersView(c echo.Context) error {
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
	limit := 1000
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			return c.String(http.StatusBadRequest, "Invalid limit (must be 1-100)")
		}
		limit = parsedLimit
	}

	var workers []*model.Worker
	var err error
	if search != "" {
		workers, err = helper.Queuer.GetWorkersBySearch(search, lastId, limit)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to search workers")
		}
	} else {
		workers, err = helper.Queuer.GetWorkers(lastId, limit)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to retrieve workers")
		}
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/workers?search=%s&limit=%d&lastId=%d", search, limit, lastId))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.Workers(workers, search))
}

// StopWorkersView handles stopping workers
func (m *ManagerHandler) StopWorkersView(c echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if !ok || len(ridStrings) == 0 {
		return renderPopupOrJson(c, http.StatusBadRequest, "No worker RIDs provided")
	}

	var rids []uuid.UUID
	for _, ridStr := range ridStrings {
		rid, err := uuid.Parse(ridStr)
		if err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid worker RID: %s", ridStr))
		}
		rids = append(rids, rid)
	}

	// Stop each worker
	for _, rid := range rids {
		err := helper.Queuer.StopWorker(rid)
		if err != nil {
			return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to stop worker %s: %v", rid, err))
		}
	}

	return renderPopupOrJson(c, http.StatusOK, fmt.Sprintf("Successfully requested stop for %d worker(s)", len(rids)))
}

// StopWorkersGracefullyView handles gracefully stopping workers
func (m *ManagerHandler) StopWorkersGracefullyView(c echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if !ok || len(ridStrings) == 0 {
		return renderPopupOrJson(c, http.StatusBadRequest, "No worker RIDs provided")
	}

	var rids []uuid.UUID
	for _, ridStr := range ridStrings {
		rid, err := uuid.Parse(ridStr)
		if err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid worker RID: %s", ridStr))
		}
		rids = append(rids, rid)
	}

	// Gracefully stop each worker
	for _, rid := range rids {
		err := helper.Queuer.StopWorkerGracefully(rid)
		if err != nil {
			return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to gracefully stop worker %s: %v", rid, err))
		}
	}

	return renderPopupOrJson(c, http.StatusOK, fmt.Sprintf("Successfully requested graceful stop for %d worker(s)", len(rids)))
}
