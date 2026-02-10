package database

import (
	"fmt"
	"manager/model"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/siherrmann/queuer/helper"
	vm "github.com/siherrmann/validator/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskNewTaskDBHandler(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}

	t.Run("Valid call NewTaskDBHandler", func(t *testing.T) {
		database := helper.NewTestDatabase(dbConfig)

		taskDbHandler, err := NewTaskDBHandler(database, true)
		assert.NoError(t, err, "Expected NewTaskDBHandler to not return an error")
		require.NotNil(t, taskDbHandler, "Expected NewTaskDBHandler to return a non-nil instance")
		require.NotNil(t, taskDbHandler.db, "Expected NewTaskDBHandler to have a non-nil database instance")
		require.NotNil(t, taskDbHandler.db.Instance, "Expected NewTaskDBHandler to have a non-nil database connection instance")

		exists, err := taskDbHandler.CheckTableExistance()
		assert.NoError(t, err)
		assert.True(t, exists)

		err = taskDbHandler.DropTable()
		assert.NoError(t, err)
	})

	t.Run("Invalid call NewTaskDBHandler with nil database", func(t *testing.T) {
		_, err := NewTaskDBHandler(nil, true)
		assert.Error(t, err, "Expected error when creating TaskDBHandler with nil database")
		assert.Contains(t, err.Error(), "database connection is nil", "Expected specific error message for nil database connection")
	})
}

func TestTaskCheckTableExistance(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	exists, err := taskDbHandler.CheckTableExistance()
	assert.NoError(t, err, "Expected CheckTableExistance to not return an error")
	assert.True(t, exists, "Expected task table to exist")
}

func TestTaskCreateTable(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	err = taskDbHandler.CreateTable()
	assert.NoError(t, err, "Expected CreateTable to not return an error")
}

func TestTaskDropTable(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	err = taskDbHandler.DropTable()
	assert.NoError(t, err, "Expected DropTable to not return an error")
}

func TestTaskInsertTask(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	task := &model.Task{
		Key:         "test_task",
		Name:        "Test Task",
		Description: "This is a test task",
		InputParameters: []vm.Validation{
			{Key: "input", Type: vm.String, Requirement: "min1"},
			{Key: "count", Type: vm.Int, Requirement: "min1"},
		},
	}

	insertedTask, err := taskDbHandler.InsertTask(task)
	assert.NoError(t, err, "Expected InsertTask to not return an error")
	assert.NotNil(t, insertedTask, "Expected InsertTask to return a non-nil task")
	assert.Equal(t, task.Key, insertedTask.Key, "Expected task key to match")
	assert.Equal(t, task.Name, insertedTask.Name, "Expected task name to match")
	assert.Equal(t, task.Description, insertedTask.Description, "Expected task description to match")
	assert.NotEqual(t, uuid.Nil, insertedTask.RID, "Expected task RID to be generated")
	assert.Greater(t, insertedTask.ID, 0, "Expected task ID to be greater than 0")
	assert.Len(t, insertedTask.InputParameters, 2, "Expected 2 input parameters")
	assert.WithinDuration(t, insertedTask.CreatedAt, time.Now(), 1*time.Second, "Expected inserted task CreatedAt time to match")
	assert.WithinDuration(t, insertedTask.UpdatedAt, time.Now(), 1*time.Second, "Expected inserted task UpdatedAt time to match")
}

func TestTaskUpdateTask(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	// Insert initial task
	task := &model.Task{
		Key:         "test_task_update",
		Name:        "Test Task Update",
		Description: "Original description",
		InputParameters: []vm.Validation{
			{Key: "input", Type: vm.String, Requirement: "min1"},
		},
	}

	insertedTask, err := taskDbHandler.InsertTask(task)
	require.NoError(t, err, "Expected InsertTask to not return an error")

	// Update the task
	insertedTask.Key = "updated_task_key"
	insertedTask.Name = "Updated Task"
	insertedTask.Description = "Updated description"
	insertedTask.InputParameters = []vm.Validation{
		{Key: "input", Type: vm.String, Requirement: "min1"},
		{Key: "output", Type: vm.String, Requirement: "min1"},
	}

	updatedTask, err := taskDbHandler.UpdateTask(insertedTask)
	assert.NoError(t, err, "Expected UpdateTask to not return an error")
	assert.NotNil(t, updatedTask, "Expected UpdateTask to return a non-nil task")
	assert.Equal(t, "updated_task_key", updatedTask.Key, "Expected updated task key to match")
	assert.Equal(t, "Updated Task", updatedTask.Name, "Expected updated task name to match")
	assert.Equal(t, "Updated description", updatedTask.Description, "Expected updated task description to match")
	assert.Len(t, updatedTask.InputParameters, 2, "Expected 2 input parameters after update")
	assert.True(t, updatedTask.UpdatedAt.After(insertedTask.UpdatedAt), "Expected UpdatedAt to be after original")
}

func TestTaskUpdateTaskNonExistent(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	task := &model.Task{
		RID:  uuid.New(),
		Name: "non_existent",
		InputParameters: []vm.Validation{
			{Key: "input", Type: vm.String, Requirement: "min1"},
		},
	}

	_, err = taskDbHandler.UpdateTask(task)
	assert.Error(t, err, "Expected UpdateTask to return an error for non-existent task")
	assert.Contains(t, err.Error(), "task not found", "Expected error message to contain 'task not found'")
}

func TestTaskDeleteTask(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	// Insert task
	task := &model.Task{
		Key:  "test_task_delete",
		Name: "Test Task Delete",
		InputParameters: []vm.Validation{
			{Key: "input", Type: vm.String, Requirement: "min1"},
		},
	}

	insertedTask, err := taskDbHandler.InsertTask(task)
	require.NoError(t, err, "Expected InsertTask to not return an error")

	// Delete the task
	err = taskDbHandler.DeleteTask(insertedTask.RID)
	assert.NoError(t, err, "Expected DeleteTask to not return an error")

	// Verify task is deleted
	_, err = taskDbHandler.SelectTask(insertedTask.RID)
	assert.Error(t, err, "Expected SelectTask to return an error for deleted task")
}

func TestTaskDeleteTaskNonExistent(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	err = taskDbHandler.DeleteTask(uuid.New())
	assert.Error(t, err, "Expected DeleteTask to return an error for non-existent task")
	assert.Contains(t, err.Error(), "task not found", "Expected error message to contain 'task not found'")
}

func TestTaskSelectTask(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	// Insert task
	task := &model.Task{
		Name: "test_task_select",
		InputParameters: []vm.Validation{
			{Key: "input", Type: vm.String, Requirement: "min1"},
			{Key: "count", Type: vm.Int, Requirement: "min0"},
		},
	}

	insertedTask, err := taskDbHandler.InsertTask(task)
	require.NoError(t, err, "Expected InsertTask to not return an error")

	// Select the task
	selectedTask, err := taskDbHandler.SelectTask(insertedTask.RID)
	assert.NoError(t, err, "Expected SelectTask to not return an error")
	assert.NotNil(t, selectedTask, "Expected SelectTask to return a non-nil task")
	assert.Equal(t, insertedTask.RID, selectedTask.RID, "Expected task RID to match")
	assert.Equal(t, insertedTask.Name, selectedTask.Name, "Expected task name to match")
	assert.Len(t, selectedTask.InputParameters, 2, "Expected 2 input parameters")
}

func TestTaskSelectTaskByKey(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	// Insert task
	task := &model.Task{
		Key:  "test_task_by_key",
		Name: "Test Task By Key",
		InputParameters: []vm.Validation{
			{Key: "input", Type: vm.String, Requirement: "min1"},
		},
	}

	insertedTask, err := taskDbHandler.InsertTask(task)
	require.NoError(t, err, "Expected InsertTask to not return an error")

	// Select the task by key
	selectedTask, err := taskDbHandler.SelectTaskByKey(task.Key)
	assert.NoError(t, err, "Expected SelectTaskByKey to not return an error")
	assert.NotNil(t, selectedTask, "Expected SelectTaskByKey to return a non-nil task")
	assert.Equal(t, insertedTask.RID, selectedTask.RID, "Expected task RID to match")
	assert.Equal(t, insertedTask.Key, selectedTask.Key, "Expected task key to match")
}

func TestTaskSelectAllTasks(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	// Insert multiple tasks
	taskCount := 5
	for i := 0; i < taskCount; i++ {
		task := &model.Task{
			Key:  fmt.Sprintf("test_task_%d", i),
			Name: fmt.Sprintf("Test Task %d", i),
			InputParameters: []vm.Validation{
				{Key: "input", Type: vm.String, Requirement: "min1"},
			},
		}
		_, err := taskDbHandler.InsertTask(task)
		require.NoError(t, err, "Expected InsertTask to not return an error")
	}

	// Select all tasks
	tasks, err := taskDbHandler.SelectAllTasks(0, 10)
	assert.NoError(t, err, "Expected SelectAllTasks to not return an error")
	assert.Len(t, tasks, taskCount, "Expected to retrieve all inserted tasks")
}

func TestTaskSelectAllTasksWithPagination(t *testing.T) {
	helper.SetTestDatabaseConfigEnvs(t, dbPort)
	dbConfig, err := helper.NewDatabaseConfiguration()
	if err != nil {
		t.Fatalf("failed to create database configuration: %v", err)
	}
	database := helper.NewTestDatabase(dbConfig)

	taskDbHandler, err := NewTaskDBHandler(database, true)
	require.NoError(t, err, "Expected NewTaskDBHandler to not return an error")

	// Insert multiple tasks
	taskCount := 10
	for i := 0; i < taskCount; i++ {
		task := &model.Task{
			Key:  fmt.Sprintf("pagination_task_%d", i),
			Name: fmt.Sprintf("Pagination Task %d", i),
			InputParameters: []vm.Validation{
				{Key: "input", Type: vm.String, Requirement: "min1"},
			},
		}
		_, err := taskDbHandler.InsertTask(task)
		require.NoError(t, err, "Expected InsertTask to not return an error")
	}

	// First page
	firstPage, err := taskDbHandler.SelectAllTasks(0, 5)
	assert.NoError(t, err, "Expected SelectAllTasks to not return an error")
	assert.Len(t, firstPage, 5, "Expected first page to have 5 tasks")

	// Second page
	lastID := firstPage[len(firstPage)-1].ID
	secondPage, err := taskDbHandler.SelectAllTasks(lastID, 5)
	assert.NoError(t, err, "Expected SelectAllTasks to not return an error")
	assert.Len(t, secondPage, 5, "Expected second page to have 5 tasks")

	// Verify no overlap
	assert.NotEqual(t, firstPage[0].ID, secondPage[0].ID, "Expected different tasks in different pages")
}
