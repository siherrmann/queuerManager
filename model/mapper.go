package model

import (
	"fmt"

	"github.com/google/uuid"
)

type Mapper interface {
	IsEmpty() bool
	ToKey() string
	ToName() string
	ToIdentifier() string
	ToDataMap() *DataMap
	ToData() []UniversalSubMapper
}

type UniversalMapper struct {
	Data []UniversalSubMapper
}

type UniversalSubMapper struct {
	Key      string
	Data     any
	Link     string
	ViewType string
}

func (u UniversalMapper) IsEmpty() bool {
	return len(u.Data) == 0
}

func (u UniversalMapper) ToKey() string {
	return u.ToName()
}

func (u UniversalMapper) ToName() string {
	if len(u.Data) > 0 {
		return fmt.Sprint(u.Data[0].Data)
	}
	return ""
}

func (u UniversalMapper) ToIdentifier() string {
	if rid, ok := u.Data[0].Data.(uuid.UUID); ok {
		return rid.String()
	} else if ridStr, ok := u.Data[0].Data.(string); ok {
		return ridStr
	}
	return ""
}

func (u UniversalMapper) ToDataMap() *DataMap {
	dataMap := DataMap{}
	for _, item := range u.Data {
		dataMap[fmt.Sprint(item.Key)] = item.Data
	}
	return &dataMap
}

func (u UniversalMapper) ToData() []UniversalSubMapper {
	return u.Data
}
