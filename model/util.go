package model

import "github.com/google/uuid"

type KeyValuePair struct {
	Key   string
	Value string
}

type TableRows []TableRow

type TableRow struct {
	RID uuid.UUID
	Row Mapper
}
