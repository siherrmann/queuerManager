package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/siherrmann/queuerManager/model"

	"github.com/google/uuid"
	"github.com/siherrmann/queuer/helper"
	vm "github.com/siherrmann/validator/model"
)

// TaskDBHandlerFunctions defines the interface for Task database operations.
type TaskDBHandlerFunctions interface {
	CheckTableExistance() (bool, error)
	CreateTable() error
	DropTable() error
	InsertTask(task *model.Task) (*model.Task, error)
	UpdateTask(task *model.Task) (*model.Task, error)
	DeleteTask(rid uuid.UUID) error
	SelectTask(rid uuid.UUID) (*model.Task, error)
	SelectTaskByKey(key string) (*model.Task, error)
	SelectAllTasks(lastID int, entries int) ([]*model.Task, error)
	SelectAllTasksBySearch(search string, lastID int, entries int) ([]*model.Task, error)
}

// TaskDBHandler implements TaskDBHandlerFunctions and holds the database connection.
type TaskDBHandler struct {
	db *helper.Database
}

// NewTaskDBHandler creates a new instance of TaskDBHandler.
// It initializes the database connection and optionally drops existing tables.
// If withTableDrop is true, it will drop the existing task table before creating a new one
func NewTaskDBHandler(dbConnection *helper.Database, withTableDrop bool) (*TaskDBHandler, error) {
	if dbConnection == nil {
		return nil, helper.NewError("database connection validation", fmt.Errorf("database connection is nil"))
	}

	taskDbHandler := &TaskDBHandler{
		db: dbConnection,
	}

	if withTableDrop {
		err := taskDbHandler.DropTable()
		if err != nil {
			return nil, helper.NewError("drop table", err)
		}
	}

	err := taskDbHandler.CreateTable()
	if err != nil {
		return nil, helper.NewError("create table", err)
	}

	return taskDbHandler, nil
}

// CheckTableExistance checks if the 'task' table exists in the database.
// It returns true if the table exists, otherwise false.
func (r TaskDBHandler) CheckTableExistance() (bool, error) {
	taskExists, err := r.db.CheckTableExistance("task")
	if err != nil {
		return false, helper.NewError("task table", err)
	}
	return taskExists, nil
}

// CreateTable creates the 'task' table in the database.
// If the table already exists, it does not create it again.
func (r TaskDBHandler) CreateTable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		CREATE TABLE IF NOT EXISTS task (
			id SERIAL PRIMARY KEY,
			rid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
			key VARCHAR(100) UNIQUE NOT NULL,
			name VARCHAR(120) DEFAULT '',
			description TEXT DEFAULT '',
			input_parameters JSONB NOT NULL DEFAULT '[]'::jsonb,
			input_parameters_keyed JSONB NOT NULL DEFAULT '[]'::jsonb,
			output_parameters JSONB NOT NULL DEFAULT '[]'::jsonb,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_task_rid ON task(rid);
		CREATE INDEX IF NOT EXISTS idx_task_name ON task(name);
	`

	_, err := r.db.Instance.ExecContext(ctx, query)
	if err != nil {
		return helper.NewError("create task table", err)
	}

	r.db.Logger.Info("Checked/created table task")

	return nil
}

// DropTable drops the 'task' table from the database.
func (r TaskDBHandler) DropTable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `DROP TABLE IF EXISTS task`
	_, err := r.db.Instance.ExecContext(ctx, query)
	if err != nil {
		return helper.NewError("drop task table", err)
	}

	r.db.Logger.Info("Dropped table task")

	return nil
}

// InsertTask inserts a new task record into the database.
func (r TaskDBHandler) InsertTask(task *model.Task) (*model.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	input_parametersJSON, err := json.Marshal(task.InputParameters)
	if err != nil {
		return nil, helper.NewError("marshal input_parameters", err)
	}

	input_parametersKeyedJSON, err := json.Marshal(task.InputParametersKeyed)
	if err != nil {
		return nil, helper.NewError("marshal input_parameters_keyed", err)
	}

	outputParametersJSON, err := json.Marshal(task.OutputParameters)
	if err != nil {
		return nil, helper.NewError("marshal output_parameters", err)
	}

	newTask := &model.Task{}
	query := `
		INSERT INTO task (
			key,
			name,
			description,
			input_parameters,
			input_parameters_keyed,
			output_parameters
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			rid,
			key,
			name,
			description,
			input_parameters,
			input_parameters_keyed,
			output_parameters,
			created_at,
			updated_at`

	var input_parametersData []byte
	var input_parametersKeyedData []byte
	var outputParametersData []byte
	err = r.db.Instance.QueryRowContext(ctx, query, task.Key, task.Name, task.Description, input_parametersJSON, input_parametersKeyedJSON, outputParametersJSON).Scan(
		&newTask.ID,
		&newTask.RID,
		&newTask.Key,
		&newTask.Name,
		&newTask.Description,
		&input_parametersData,
		&input_parametersKeyedData,
		&outputParametersData,
		&newTask.CreatedAt,
		&newTask.UpdatedAt,
	)
	if err != nil {
		return nil, helper.NewError("insert task", err)
	}

	err = json.Unmarshal(input_parametersData, &newTask.InputParameters)
	if err != nil {
		return nil, helper.NewError("unmarshal input_parameters", err)
	}

	err = json.Unmarshal(input_parametersKeyedData, &newTask.InputParametersKeyed)
	if err != nil {
		return nil, helper.NewError("unmarshal input_parameters_keyed", err)
	}

	err = json.Unmarshal(outputParametersData, &newTask.OutputParameters)
	if err != nil {
		return nil, helper.NewError("unmarshal output_parameters", err)
	}

	return newTask, nil
}

// UpdateTask updates an existing task record in the database.
func (r TaskDBHandler) UpdateTask(task *model.Task) (*model.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	input_parametersJSON, err := json.Marshal(task.InputParameters)
	if err != nil {
		return nil, helper.NewError("marshal input_parameters", err)
	}

	input_parametersKeyedJSON, err := json.Marshal(task.InputParametersKeyed)
	if err != nil {
		return nil, helper.NewError("marshal input_parameters_keyed", err)
	}

	outputParametersJSON, err := json.Marshal(task.OutputParameters)
	if err != nil {
		return nil, helper.NewError("marshal output_parameters", err)
	}

	updatedTask := &model.Task{}
	query := `
		UPDATE task
		SET
			key = $1,
			name = $2,
			description = $3,
			input_parameters = $4,
			input_parameters_keyed = $5,
			output_parameters = $6,
			updated_at = NOW()
		WHERE rid = $7
		RETURNING
			id,
			rid,
			key,
			name,
			description,
			input_parameters,
			input_parameters_keyed,
			output_parameters,
			created_at,
			updated_at`

	var input_parametersData []byte
	var input_parametersKeyedData []byte
	var outputParametersData []byte
	err = r.db.Instance.QueryRowContext(ctx, query, task.Key, task.Name, task.Description, input_parametersJSON, input_parametersKeyedJSON, outputParametersJSON, task.RID).Scan(
		&updatedTask.ID,
		&updatedTask.RID,
		&updatedTask.Key,
		&updatedTask.Name,
		&updatedTask.Description,
		&input_parametersData,
		&input_parametersKeyedData,
		&outputParametersData,
		&updatedTask.CreatedAt,
		&updatedTask.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, helper.NewError("task not found", fmt.Errorf("no task with rid %s", task.RID))
		}
		return nil, helper.NewError("update task", err)
	}

	err = json.Unmarshal(input_parametersData, &updatedTask.InputParameters)
	if err != nil {
		return nil, helper.NewError("unmarshal input_parameters", err)
	}

	err = json.Unmarshal(input_parametersKeyedData, &updatedTask.InputParametersKeyed)
	if err != nil {
		return nil, helper.NewError("unmarshal input_parameters_keyed", err)
	}

	err = json.Unmarshal(outputParametersData, &updatedTask.OutputParameters)
	if err != nil {
		return nil, helper.NewError("unmarshal output_parameters", err)
	}

	return updatedTask, nil
}

// DeleteTask deletes a task record from the database by RID.
func (r TaskDBHandler) DeleteTask(rid uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `DELETE FROM task WHERE rid = $1`
	result, err := r.db.Instance.ExecContext(ctx, query, rid)
	if err != nil {
		return helper.NewError("delete task", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return helper.NewError("get rows affected", err)
	}

	if rowsAffected == 0 {
		return helper.NewError("task not found", fmt.Errorf("no task with rid %s", rid))
	}

	return nil
}

// SelectTask retrieves a task by RID from the database.
func (r TaskDBHandler) SelectTask(rid uuid.UUID) (*model.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	task := &model.Task{}
	query := `
		SELECT
			id,
			rid,
			key,
			name,
			description,
			input_parameters,
			input_parameters_keyed,
			output_parameters,
			created_at,
			updated_at
		FROM task
		WHERE rid = $1
	`

	var input_parametersData []byte
	var input_parametersKeyedData []byte
	var outputParametersData []byte
	err := r.db.Instance.QueryRowContext(ctx, query, rid).Scan(
		&task.ID,
		&task.RID,
		&task.Key,
		&task.Name,
		&task.Description,
		&input_parametersData,
		&input_parametersKeyedData,
		&outputParametersData,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, helper.NewError("task not found", fmt.Errorf("no task with rid %s", rid))
		}
		return nil, helper.NewError("select task", err)
	}

	err = json.Unmarshal(input_parametersData, &task.InputParameters)
	if err != nil {
		return nil, helper.NewError("unmarshal input_parameters", err)
	}

	err = json.Unmarshal(input_parametersKeyedData, &task.InputParametersKeyed)
	if err != nil {
		return nil, helper.NewError("unmarshal input_parameters_keyed", err)
	}

	err = json.Unmarshal(outputParametersData, &task.OutputParameters)
	if err != nil {
		return nil, helper.NewError("unmarshal output_parameters", err)
	}

	return task, nil
}

// SelectTaskByKey retrieves a task by key from the database.
func (r TaskDBHandler) SelectTaskByKey(key string) (*model.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	task := &model.Task{}
	query := `
		SELECT id, rid, key, name, description, input_parameters, input_parameters_keyed, output_parameters, created_at, updated_at
		FROM task
		WHERE key = $1
	`

	var input_parametersData []byte
	var input_parametersKeyedData []byte
	var outputParametersData []byte
	err := r.db.Instance.QueryRowContext(ctx, query, key).Scan(
		&task.ID,
		&task.RID,
		&task.Key,
		&task.Name,
		&task.Description,
		&input_parametersData,
		&input_parametersKeyedData,
		&outputParametersData,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, helper.NewError("task not found", fmt.Errorf("no task with key %s", key))
		}
		return nil, helper.NewError("select task by key", err)
	}

	err = json.Unmarshal(input_parametersData, &task.InputParameters)
	if err != nil {
		return nil, helper.NewError("unmarshal input_parameters", err)
	}

	err = json.Unmarshal(input_parametersKeyedData, &task.InputParametersKeyed)
	if err != nil {
		return nil, helper.NewError("unmarshal input_parameters_keyed", err)
	}

	err = json.Unmarshal(outputParametersData, &task.OutputParameters)
	if err != nil {
		return nil, helper.NewError("unmarshal output_parameters", err)
	}

	return task, nil
}

// SelectAllTasks retrieves all tasks from the database with pagination.
// lastID is the ID of the last task from the previous page (0 for first page)
// entries is the maximum number of tasks to return
func (r TaskDBHandler) SelectAllTasks(lastID int, entries int) ([]*model.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT
			id,
			rid,
			key,
			name,
			description,
			input_parameters,
			input_parameters_keyed,
			output_parameters,
			created_at,
			updated_at
		FROM task
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2
	`

	rows, err := r.db.Instance.QueryContext(ctx, query, lastID, entries)
	if err != nil {
		return nil, helper.NewError("select all tasks", err)
	}
	defer rows.Close()

	tasks := []*model.Task{}
	for rows.Next() {
		task := &model.Task{}
		var input_parametersData []byte
		var input_parametersKeyedData []byte
		var outputParametersData []byte

		err := rows.Scan(
			&task.ID,
			&task.RID,
			&task.Key,
			&task.Name,
			&task.Description,
			&input_parametersData,
			&input_parametersKeyedData,
			&outputParametersData,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, helper.NewError("scan task", err)
		}

		err = json.Unmarshal(input_parametersData, &task.InputParameters)
		if err != nil {
			log.Printf("Warning: failed to unmarshal input_parameters for task %s: %v", task.RID, err)
			task.InputParameters = []vm.Validation{}
		}

		err = json.Unmarshal(input_parametersKeyedData, &task.InputParametersKeyed)
		if err != nil {
			log.Printf("Warning: failed to unmarshal input_parameters_keyed for task %s: %v", task.RID, err)
			task.InputParametersKeyed = []vm.Validation{}
		}

		err = json.Unmarshal(outputParametersData, &task.OutputParameters)
		if err != nil {
			log.Printf("Warning: failed to unmarshal output_parameters for task %s: %v", task.RID, err)
			task.OutputParameters = []vm.Validation{}
		}

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, helper.NewError("rows iteration", err)
	}

	return tasks, nil
}

// SelectAllTasksBySearch retrieves tasks matching the search query with pagination.
// search is the search string to match against rid, key, name, and description
// lastID is the ID of the last task from the previous page (0 for first page)
// entries is the maximum number of tasks to return
func (r TaskDBHandler) SelectAllTasksBySearch(search string, lastID int, entries int) ([]*model.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.Instance.QueryContext(ctx,
		`SELECT
			id,
			rid,
			key,
			name,
			description,
			input_parameters,
			input_parameters_keyed,
			output_parameters,
			created_at,
			updated_at
		FROM task
		WHERE (task.rid::text ILIKE '%' || $1 || '%'
				OR task.key ILIKE '%' || $1 || '%'
				OR task.name ILIKE '%' || $1 || '%'
				OR task.description ILIKE '%' || $1 || '%')
			AND (0 = $2
				OR task.created_at < (
					SELECT t.created_at
					FROM task AS t
					WHERE t.id = $2))
		ORDER BY task.created_at DESC
		LIMIT $3
		`,
		search,
		lastID,
		entries,
	)
	if err != nil {
		return nil, helper.NewError("select tasks by search", err)
	}
	defer rows.Close()

	tasks := []*model.Task{}
	for rows.Next() {
		task := &model.Task{}
		var input_parametersData []byte
		var input_parametersKeyedData []byte
		var outputParametersData []byte

		err := rows.Scan(
			&task.ID,
			&task.RID,
			&task.Key,
			&task.Name,
			&task.Description,
			&input_parametersData,
			&input_parametersKeyedData,
			&outputParametersData,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, helper.NewError("scan task", err)
		}

		err = json.Unmarshal(input_parametersData, &task.InputParameters)
		if err != nil {
			log.Printf("Warning: failed to unmarshal input_parameters for task %s: %v", task.RID, err)
			task.InputParameters = []vm.Validation{}
		}

		err = json.Unmarshal(input_parametersKeyedData, &task.InputParametersKeyed)
		if err != nil {
			log.Printf("Warning: failed to unmarshal input_parameters_keyed for task %s: %v", task.RID, err)
			task.InputParametersKeyed = []vm.Validation{}
		}

		err = json.Unmarshal(outputParametersData, &task.OutputParameters)
		if err != nil {
			log.Printf("Warning: failed to unmarshal output_parameters for task %s: %v", task.RID, err)
			task.OutputParameters = []vm.Validation{}
		}

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, helper.NewError("rows iteration", err)
	}

	return tasks, nil
}
