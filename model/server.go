package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type ServerFilter struct {
	Name    *string `json:"name"`     // 名字完全匹配，支持*
	IpAddr  *string `json:"ip_addr"`  // IP 地址完全匹配，支持* 匹配所有
	EnvType *string `json:"env_type"` // 机器 Tags 中的 EnvType，支持* 匹配所有
	Team    *string `json:"team"`     // 机器 Tags 中的 Team，支持* 匹配所有
}

func (a ServerFilter) ToV1() *ServerFilterV1 {
	res := &ServerFilterV1{}
	if a.Name != nil {
		res.Name = []string{*a.Name}
	}
	if a.IpAddr != nil {
		res.IpAddr = []string{*a.IpAddr}
	}
	if a.EnvType != nil {
		res.EnvType = []string{*a.EnvType}
	}
	if a.Team != nil {
		res.Team = []string{*a.Team}
	}
	return res
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

type PolicyOld struct {
	ID           string       `json:"id" gorm:"column:id;primary_key;not null"`
	CreatedAt    time.Time    `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time    `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted    bool         `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	Name         string       `json:"name" gorm:"column:name;not null"`
	Users        ArrayString  `json:"users" gorm:"column:users;type:json;not null"`
	ServerFilter ServerFilter `json:"server_filter" gorm:"column:server_filter;type:json;not null"`
	Actions      ArrayString  `json:"actions" gorm:"column:actions;type:json;not null"`
	ExpiresAt    time.Time    `json:"expires_at" gorm:"column:expires_at;not null"`
	Approver     string       `json:"approver" gorm:"column:approver"`       // 审批人
	ApprovalID   string       `json:"approval_id" gorm:"column:approval_id"` // 审批ID
	IsEnabled    bool         `json:"is_enabled" gorm:"column:is_enabled;default:false;not null"`
}

func (PolicyOld) TableName() string {
	return "jms_go_policy"
}
