package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"manager/helper"
	"manager/view/screens"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/siherrmann/queuer/model"
)

// =======API Handlers=======

// AddJob handles the addition of a new job
func (m *ManagerHandler) AddJob(c echo.Context) error {
	taskKey := c.Param("taskKey")
	task, err := m.taskDB.SelectTaskByKey(taskKey)
	if err != nil {
		return c.String(http.StatusNotFound, "Task not found")
	}

	// Validate regular parameters
	parameters := map[string]any{}
	validations := task.InputParameters
	validations = append(validations, task.InputParametersKeyed...)
	err = m.validator.UnmapOrUnmarshalValidateAndUpdateWithValidation(c.Request(), &parameters, validations)
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Validation error: %v", err))
	}

	// Validate keyed parameters (extract from form values with "keyed_" prefix)
	parametersList := []any{}
	parametersKeyed := map[string]any{}
	for _, v := range task.InputParameters {
		if val, ok := parameters[v.Key]; ok {
			parametersList = append(parametersList, val)
		}
	}
	for _, v := range task.InputParametersKeyed {
		if val, ok := parameters[v.Key]; ok {
			parametersKeyed[v.Key] = val
		}
	}

	// Add job with keyed parameters map and spread parameter list
	jobAdded, err := helper.Queuer.AddJob(taskKey, parametersKeyed, parametersList...)
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to add job: %v", err))
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/job?rid=%s", jobAdded.RID.String()))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.Job(jobAdded))
}

// GetJobs retrieves a paginated list of jobs
func (m *ManagerHandler) GetJobs(c echo.Context) error {
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

	jobs, err := helper.Queuer.GetJobs(lastId, limit)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve jobs")
	}

	return c.JSON(http.StatusOK, jobs)
}

// CancelJob cancels a specific job by RID
func (m *ManagerHandler) CancelJob(c echo.Context) error {
	ridStr := c.Param("rid")
	rid, err := uuid.Parse(ridStr)
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, "Invalid job RID format")
	}

	cancelledJob, err := helper.Queuer.CancelJob(rid)
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, "Failed to cancel job")
	}

	return renderPopupOrJson(c, http.StatusOK, cancelledJob)
}

// CancelJobs cancels multiple jobs by their RIDs
func (m *ManagerHandler) CancelJobs(c echo.Context) error {
	form, err := c.FormParams()
	if _, ok := form["rid"]; !ok || err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, "Failed to parse form with job RIDs")
	}
	ridStrs := form["rid"]
	if len(ridStrs) == 0 {
		return renderPopupOrJson(c, http.StatusBadRequest, "No job RIDs provided")
	}

	var rids []uuid.UUID
	for _, ridStr := range ridStrs {
		rid, err := uuid.Parse(ridStr)
		if err != nil {
			return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid job RID format: %s", ridStr))
		}
		rids = append(rids, rid)
	}

	var cancelledJobs []*model.Job
	for _, rid := range rids {
		cancelledJob, err := helper.Queuer.CancelJob(rid)
		if err != nil {
			return renderPopupOrJson(c, http.StatusInternalServerError, "Failed to cancel jobs")
		}
		cancelledJobs = append(cancelledJobs, cancelledJob)
	}

	return renderPopupOrJson(c, http.StatusOK, fmt.Sprintf("%v jobs cancelled successfully", len(cancelledJobs)))
}

// DeleteJob deletes a specific job by RID
func (m *ManagerHandler) DeleteJob(c echo.Context) error {
	ridStr := c.Param("rid")
	rid, err := uuid.Parse(ridStr)
	if err != nil {
		return renderPopupOrJson(c, http.StatusBadRequest, fmt.Sprintf("Invalid rid: %v", err))
	}

	err = helper.Queuer.DeleteJob(rid)
	if err != nil {
		return renderPopupOrJson(c, http.StatusInternalServerError, fmt.Sprintf("Failed to delete job: %v", err))
	}

	return renderPopupOrJson(c, http.StatusOK, "Job deleted successfully")
}

// =======View Handlers=======

// JobView renders the job detail view
func (m *ManagerHandler) JobView(c echo.Context) error {
	ridStrings, ok := c.QueryParams()["rid"]
	if len(ridStrings) == 0 || !ok {
		return renderPopupOrJson(c, http.StatusBadRequest, "Missing job RID")
	}

	rid, err := uuid.Parse(ridStrings[0])
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid job RID format")
	}

	job, err := helper.Queuer.GetJob(rid)
	if err != nil {
		job, err = helper.Queuer.GetJobEnded(rid)
		if err != nil {
			return renderPopupOrJson(c, http.StatusNotFound, "Job not found")
		}
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/job?rid=%s", rid.String()))
	c.Response().Header().Add("HX-Retarget", "#body")

	status := http.StatusOK
	if job.Status == model.JobStatusFailed || job.Status == model.JobStatusCancelled || job.Status == model.JobStatusSucceeded {
		status = 286 // Custom status code to end htmx polling
	}

	return render(c, screens.Job(job), status)
}

// JobsView renders the jobs view
func (m *ManagerHandler) JobsView(c echo.Context) error {
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

	var jobs []*model.Job
	var err error
	if search != "" {
		log.Printf("searching for: %v", search)
		jobs, err = helper.Queuer.GetJobsBySearch(search, lastId, limit)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to search jobs")
		}
		log.Printf("found jobs: %v", jobs)
	} else {
		jobs, err = helper.Queuer.GetJobs(lastId, limit)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to retrieve jobs")
		}
	}

	c.Response().Header().Add("HX-Push-Url", fmt.Sprintf("/jobs?search=%s&limit=%d&lastId=%d", search, limit, lastId))
	c.Response().Header().Add("HX-Retarget", "#body")

	return render(c, screens.Jobs(jobs, search))
}
