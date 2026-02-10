package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/siherrmann/queuerManager/view/screens"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuer/model"
)

// GetJobArchive retrieves a specific archived job by RID
func (m *ManagerHandler) GetJobArchive(c *echo.Context) error {
	ridStr := c.Param("rid")
	rid, err := uuid.Parse(ridStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid job archive RID format")
	}

	job, err := m.Queuer.GetJobEnded(rid)
	if err != nil {
		return c.String(http.StatusNotFound, "Archived job not found")
	}

	return c.JSON(http.StatusOK, job)
}

// GetJobsArchive retrieves a paginated list of archived jobs
func (m *ManagerHandler) GetJobsArchive(c *echo.Context) error {
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

	jobArchives, err := m.Queuer.GetJobsEnded(lastId, limit)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve archived jobs")
	}

	return c.JSON(http.StatusOK, jobArchives)
}

// ======View Handlers======

// JobArchiveView renders the job archive view
func (m *ManagerHandler) JobArchiveView(c *echo.Context) error {
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

	var archivedJobs []*model.Job
	var err error
	if search != "" {
		archivedJobs, err = m.Queuer.GetJobsEndedBySearch(search, lastId, limit)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to search archived jobs")
		}
	} else {
		archivedJobs, err = m.Queuer.GetJobsEnded(lastId, limit)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to retrieve archived jobs")
		}
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/jobArchive?search=%s&limit=%d&lastId=%d", search, limit, lastId))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.JobArchive(archivedJobs, search))
}

// ReaddJobFromArchiveView readds a job from the archive back to the queue
func (m *ManagerHandler) ReaddJobFromArchiveView(c *echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if !ok || len(ridStrings) == 0 {
		return renderPopupOrJson(c, http.StatusBadRequest, "No job RID provided")
	}
	if len(ridStrings) > 1 {
		return renderPopupOrJson(c, http.StatusBadRequest, "Please select exactly one job")
	}

	rid, err := uuid.Parse(ridStrings[0])
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid job RID: %v", err))
	}

	readdedJob, err := m.Queuer.ReaddJobFromArchive(rid)
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to re-add job: %v", err))
	}

	return renderPopupOrJson(c, http.StatusOK, fmt.Sprintf("Job %s re-added to queue", readdedJob.RID.String()))
}
