package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
)

// 用来存储json数组，gorm默认不支持

type ArrayString []interface{}

func (a ArrayString) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ArrayString) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}

func (a ArrayString) Contains(value string) bool {
	for _, item := range a {
		if item == value || strings.Contains(item.(string), "*") {
			return true
		}
	}
	return false
}

// 可以预定义一些资产用来快速分配给其他策略c
type ServerFilter struct {
	Name    *string `json:"name" `
	IpAddr  *string `json:"ip_addr" `
	EnvType *string `json:"env_type"`
	Team    *string `json:"team"`
}

func (a ServerFilter) ToString() string {
	return fmt.Sprintf("Name:%s IP:%s Env:%s Team:%s",
		tea.StringValue(a.Name), tea.StringValue(a.IpAddr), tea.StringValue(a.EnvType), tea.StringValue(a.Team))
}

func (a ServerFilter) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ServerFilter) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}
