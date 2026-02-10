package main

import (
	"manager/handler"
	mw "manager/middleware"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SetupRoutes configures all API routes for the manager service
func SetupRoutes(e *echo.Echo, h *handler.ManagerHandler) {
	// Middleware
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Custom Middleware
	m := mw.NewMiddleware()
	e.Use(m.CsrfMiddleware())
	e.Use(m.RequestContextMiddleware)

	// View routes
	e.GET("/health", h.HealthCheck)
	e.GET("/", h.AddJobView)
	e.GET("/task/:taskKey", h.AddJobConfigView)

	e.GET("/files", h.FilesView)
	e.GET("/file", h.FileView)
	e.GET("/file/addFilePopup", h.AddFilePopupView)
	e.GET("/file/deleteFilePopup", h.DeleteFilePopupView)

	e.GET("/job", h.JobView)
	e.GET("/jobs", h.JobsView)
	e.GET("/jobArchive", h.JobArchiveView)
	e.GET("/jobArchive/readdJob", h.ReaddJobFromArchiveView)

	e.GET("/worker", h.WorkerView)
	e.GET("/workers", h.WorkersView)
	e.GET("/worker/stopWorkers", h.StopWorkersView)
	e.GET("/worker/stopWorkersGracefully", h.StopWorkersGracefullyView)

	e.GET("/tasks", h.TasksView)
	e.GET("/task", h.TaskView)
	e.GET("/task/addTaskPopup", h.AddTaskPopupView)
	e.GET("/task/updateTaskPopup", h.UpdateTaskPopupView)
	e.GET("/task/deleteTaskPopup", h.DeleteTaskPopupView)
	e.GET("/task/importTaskPopup", h.ImportTaskPopupView)

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
