package queuerManager

import (
	"net/http"

	"github.com/siherrmann/queuerManager/handler"
	mw "github.com/siherrmann/queuerManager/middleware"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// SetupRoutes configures all API routes for the manager service
func SetupRoutes(e *echo.Echo, h *handler.ManagerHandler) {
	// Middleware
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))

	// Custom Middleware
	m := mw.NewMiddleware()
	e.Use(m.RequestContextMiddleware)

	// View routes
	e.GET("/health", h.HealthCheck, m.CsrfMiddleware())
	e.GET("/", h.AddJobView, m.CsrfMiddleware())
	e.GET("/task/:taskKey", h.AddJobConfigView, m.CsrfMiddleware())

	e.GET("/files", h.FilesView, m.CsrfMiddleware())
	e.GET("/file", h.FileView, m.CsrfMiddleware())
	e.GET("/file/addFilePopup", h.AddFilePopupView, m.CsrfMiddleware())
	e.GET("/file/deleteFilePopup", h.DeleteFilePopupView, m.CsrfMiddleware())

	e.GET("/job", h.JobView, m.CsrfMiddleware())
	e.GET("/jobs", h.JobsView, m.CsrfMiddleware())
	e.GET("/jobArchive", h.JobArchiveView, m.CsrfMiddleware())
	e.GET("/jobArchive/readdJob", h.ReaddJobFromArchiveView, m.CsrfMiddleware())

	e.GET("/worker", h.WorkerView, m.CsrfMiddleware())
	e.GET("/workers", h.WorkersView, m.CsrfMiddleware())
	e.GET("/worker/stopWorkers", h.StopWorkersView, m.CsrfMiddleware())
	e.GET("/worker/stopWorkersGracefully", h.StopWorkersGracefullyView, m.CsrfMiddleware())

	e.GET("/tasks", h.TasksView, m.CsrfMiddleware())
	e.GET("/task", h.TaskView, m.CsrfMiddleware())
	e.GET("/task/addTaskPopup", h.AddTaskPopupView, m.CsrfMiddleware())
	e.GET("/task/updateTaskPopup", h.UpdateTaskPopupView, m.CsrfMiddleware())
	e.GET("/task/deleteTaskPopup", h.DeleteTaskPopupView, m.CsrfMiddleware())
	e.GET("/task/importTaskPopup", h.ImportTaskPopupView, m.CsrfMiddleware())

	// API routes
	api := e.Group("/api")

	jobs := api.Group("/job")
	jobs.POST("/addJob/:taskKey", h.AddJob)
	jobs.POST("/cancelJob/:rid", h.CancelJob)
	jobs.POST("/cancelJobs", h.CancelJobs)
	jobs.POST("/deleteJob/:rid", h.DeleteJob)
	jobs.GET("/getJobs", h.GetJobs)

	jobArchives := api.Group("/jobArchive")
	jobArchives.GET("/getJob/:rid", h.GetJobArchive)
	jobArchives.GET("/getJobs", h.GetJobsArchive)

	workers := api.Group("/worker")
	workers.GET("/getWorker/:rid", h.GetWorker)
	workers.GET("/getWorkers", h.GetWorkers)

	tasks := api.Group("/task")
	tasks.POST("/addTask", h.AddTask)
	tasks.POST("/updateTask", h.UpdateTask)
	tasks.POST("/deleteTasks", h.DeleteTasks)
	tasks.GET("/getTask/:rid", h.GetTask)
	tasks.GET("/getTaskByName/:name", h.GetTaskByName)
	tasks.GET("/getTasks", h.GetTasks)
	tasks.GET("/exportTask", h.ExportTask)
	tasks.POST("/importTask", h.ImportTask)

	files := api.Group("/file")
	files.POST("/uploadFiles", h.UploadFiles)
	files.POST("/deleteFile/:filename", h.DeleteFile)
	files.POST("/deleteFiles", h.DeleteFiles)

	connections := api.Group("/connection")
	connections.GET("/getConnections", h.GetConnections)

	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	e.Static("/static/", "./view/static")
}
