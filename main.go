package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"manager/database"
	"manager/handler"
	"manager/helper"
	"manager/model"
	"manager/upload"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	qh "github.com/siherrmann/queuer/helper"
	qmodel "github.com/siherrmann/queuer/model"
)

// main is the entry point of the manager service. It initializes the manager handler,
// sets up routes, and starts the Echo server with the port from the environment variable QUEUER_MANAGER_PORT.
func main() {
	ManagerServer(helper.GetEnvOrDefault("QUEUER_MANAGER_PORT", "3000"), 1)
}

// ManagerServer initializes the manager handler, sets up routes, and starts the Echo server.
func ManagerServer(port string, maxConcurrency int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mh, err := InitManagerHandler(ctx, cancel, maxConcurrency)
	if err != nil {
		log.Fatalf("Failed to initialize manager handler: %v", err)
	}

	e := echo.New()
	SetupRoutes(e, mh)

	e.Logger.Fatal(e.Start(":" + port))

	<-ctx.Done()
	slog.Info("Shutting down manager server")
}

// InitManagerHandler creates and configures the manager handler, including initializing the queuer, setting up the filesystem, and loading tasks from a JSON file if specified.
// It returns the initialized manager handler or an error if initialization fails.
func InitManagerHandler(ctx context.Context, cancel context.CancelFunc, maxConcurrency int) (*handler.ManagerHandler, error) {
	// Create queuer instance
	helper.InitQueuer(maxConcurrency)

	// Create filesystem from environment variables
	filesystem, err := upload.CreateFilesystemFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem: %w", err)
	}

	// Logger
	opts := qh.PrettyHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
	}
	logger := slog.New(qh.NewPrettyHandler(os.Stdout, opts))

	// Initialize task database handler
	db := &qh.Database{
		Name:     "task",
		Logger:   logger,
		Instance: helper.Queuer.DB,
	}
	taskDB, err := database.NewTaskDBHandler(db, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create task database handler: %w", err)
	}

	// Load tasks from JSON file if path is provided
	taskJSONPath := helper.GetEnvOrDefault("QUEUER_MANAGER_TASK_JSON", "")
	if taskJSONPath != "" {
		if err := loadTasksFromJSON(taskJSONPath, taskDB, logger); err != nil {
			return nil, fmt.Errorf("failed to load tasks from JSON: %w", err)
		}
	}

	// Create and configure manager handler
	mh := handler.NewManagerHandler(filesystem, taskDB)

	// Start the queuer with master settings
	masterSettings := &qmodel.MasterSettings{
		MasterLockTimeout:     time.Minute * 1,
		MasterPollInterval:    time.Second * 10,
		WorkerStaleThreshold:  time.Minute * 5,
		WorkerDeleteThreshold: time.Minute * 100,
		JobStaleThreshold:     time.Minute * 10,
		JobDeleteThreshold:    time.Minute * 100,
	}
	helper.Queuer.Start(ctx, cancel, masterSettings)

	return mh, nil
}

func loadTasksFromJSON(filePath string, taskDB database.TaskDBHandlerFunctions, logger *slog.Logger) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var tasks []*model.Task
	err = json.Unmarshal(data, &tasks)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		insertedTask, err := taskDB.InsertTask(task)
		if err != nil {
			logger.Warn("Failed to insert task", "key", task.Key, "error", err)
			continue
		}
		logger.Info("Task loaded from JSON", "key", insertedTask.Key, "name", insertedTask.Name)
	}

	logger.Info("Finished loading tasks from JSON", "file", filePath, "total", len(tasks))
	return nil
}
