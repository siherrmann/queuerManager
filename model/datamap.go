package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type DataMap map[string]interface{}

func (d DataMap) StripEmpty() DataMap {
	strippedMap := DataMap{}
	for key, value := range d {
		if fmt.Sprintf("%v", value) != "0" && fmt.Sprintf("%v", value) != "" && fmt.Sprintf("%v", value) != "false" && key != "gorilla.csrf.Token" {
			strippedMap[key] = value
		}
	}
	return strippedMap
}

func (d *DataMap) GetStringByKey(key string) string {
	if value, ok := (*d)[key]; ok {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

func (d *DataMap) GetIntByKey(key string) int {
	if value, ok := (*d)[key]; ok {
		if number, ok := value.(int); ok {
			return number
		}
		return 0
	}
	return 0
}

func (d *DataMap) GetTimeByKey(key string) string {
	if value, ok := (*d)[key]; ok {
		if date, ok := value.(time.Time); ok {
			return date.Format("2006-01-02")
		}
		return "invalid time"
	}
	return "invalid time"
}

func (d *DataMap) GetListByKey(key string) []string {
	if value, ok := (*d)[key]; ok {
		if list, ok := value.([]interface{}); ok {
			listString := []string{}
			for _, v := range list {
				listString = append(listString, fmt.Sprintf("%v", v))
			}
			return listString
		}
		return []string{fmt.Sprintf("%v", value)}
	}
	return []string{}
}

func (c DataMap) Value() (driver.Value, error) {
	return c.Marshal()
}

func (c *DataMap) Scan(value interface{}) error {
	return c.Unmarshal(value)
}

func (r DataMap) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func (r *DataMap) Unmarshal(value interface{}) error {
	if s, ok := value.(DataMap); ok {
		*r = DataMap(s)
	} else {
		b, ok := value.([]byte)
		if !ok {
			return errors.New("type assertion to []byte failed")
		}
		return json.Unmarshal(b, r)
	}
	return nil
}

func (r *DataMap) Has(key string) bool {
	if _, ok := (*r)[key]; ok {
		return true
	}
	return false
}

func (r *DataMap) ToDataMapReadable() *DataMapReadable {
	dataMapReadable := DataMapReadable{}
	for k, v := range *r {
		if date, ok := v.(time.Time); ok {
			dataMapReadable[k] = date.Format("2006-01-02")
		} else if array, ok := v.([]interface{}); ok {
			as := []string{}
			for _, d := range array {
				as = append(as, fmt.Sprintf("%v", d))
			}
			s := strings.Join(as, ", ")
			dataMapReadable[k] = s
		} else {
			dataMapReadable[k] = fmt.Sprintf("%v", v)
		}
	}
	return &dataMapReadable
}

type DataMapReadable map[string]string

func (d *DataMapReadable) GetStringByKey(key string) string {
	if value, ok := (*d)[key]; ok {
		return value
	}
	return ""
}
