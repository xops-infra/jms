package config

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
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

func (a ArrayString) Contains(value string) bool {
	for _, item := range a {
		if item == value || strings.Contains(item, "*") {
			return true
		}
	}
	return false
}

// 可以预定义一些资产用来快速分配给其他策略c
type ServerFilter struct {
	Name    *string `json:"name"`     // 名字完全匹配，支持*
	IpAddr  *string `json:"ip_addr"`  // IP 地址完全匹配，支持* 匹配所有
	EnvType *string `json:"env_type"` // 机器 Tags 中的 EnvType，支持* 匹配所有
	Team    *string `json:"team"`     // 机器 Tags 中的 Team，支持* 匹配所有
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
