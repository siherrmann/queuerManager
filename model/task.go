package model

import (
	"time"

	"github.com/google/uuid"
	vm "github.com/siherrmann/validator/model"
)

// Task represents a task configuration in the database
type Task struct {
	ID                   int             `json:"id"`
	RID                  uuid.UUID       `json:"rid"`
	Key                  string          `json:"key"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	InputParameters      []vm.Validation `json:"input_parameters"`
	InputParametersKeyed []vm.Validation `json:"input_parameters_keyed"`
	OutputParameters     []vm.Validation `json:"output_parameters"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}
