package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
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

// 用来存储json数组，gorm默认不支持

type ArrayString []string

func (a ArrayString) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ArrayString) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}

// 支持*和! aaa* 写法
func (a ArrayString) Contains(findStr string) bool {
	for _, item := range a {
		if item == findStr ||
			item == "*" ||
			(strings.Contains(item, "*") && strings.HasPrefix(findStr, strings.SplitN(item, "*", 2)[0])) ||
			(strings.HasPrefix(item, "!") && strings.TrimPrefix(item, "!") != findStr) {
			return true
		}
	}
	return false
}
