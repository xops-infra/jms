package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type StringSlice []string

func (ss *StringSlice) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Scan source was not []bytes")
	}
	return json.Unmarshal(asBytes, ss)
}

func (ss StringSlice) Value() (driver.Value, error) {
	return json.Marshal(ss)
}
