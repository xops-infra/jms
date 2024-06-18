package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// 可以预定义一些资产用来快速分配给其他策略c
type ServerFilter struct {
	Name    []string `json:"name"`     // 名字完全匹配，支持*
	IpAddr  []string `json:"ip_addr"`  // IP 地址完全匹配，支持* 匹配所有
	EnvType []string `json:"env_type"` // 机器 Tags 中的 EnvType，支持* 匹配所有
	Team    []string `json:"team"`     // 机器 Tags 中的 Team，支持* 匹配所有
}

func (a ServerFilter) ToString() string {
	return fmt.Sprintf("%v", a)
}

func (a ServerFilter) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ServerFilter) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}
