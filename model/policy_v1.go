package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type PolicyRequest struct {
	Name           *string         `json:"name" binding:"required"`
	Users          ArrayString     `json:"users"`
	Actions        ArrayString     `json:"actions"`
	ServerFilterV1 *ServerFilterV1 `json:"server_filter" binding:"required"`
	ExpiresAt      *time.Time      `json:"expires_at"` // time.Time
	IsEnabled      *bool           `json:"is_enabled"`
	ApprovalID     *string         `json:"approval_id"`
}

type Policy struct {
	ID             string          `json:"id" gorm:"column:id;primary_key;not null"`
	CreatedAt      time.Time       `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time       `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted      bool            `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	Name           string          `json:"name" gorm:"column:name;not null"`
	Users          ArrayString     `json:"users" gorm:"column:users;type:json;not null"`
	ServerFilterV1 *ServerFilterV1 `json:"server_filter_v1" gorm:"column:server_filter_v1;type:json;"`
	ServerFilter   *ServerFilter   `json:"server_filter" gorm:"clumn:server_filter;type:json;"`
	Actions        ArrayString     `json:"actions" gorm:"column:actions;type:json;not null"`
	ExpiresAt      time.Time       `json:"expires_at" gorm:"column:expires_at;not null"`
	Approver       string          `json:"approver" gorm:"column:approver"`       // 审批人
	ApprovalID     string          `json:"approval_id" gorm:"column:approval_id"` // 审批ID
	IsEnabled      bool            `json:"is_enabled" gorm:"column:is_enabled;default:false;not null"`
}

func (p *Policy) IsExpired() bool {
	return time.Since(p.ExpiresAt) > 0
}

func (Policy) TableName() string {
	return "jms_go_policy"
}

// 可以预定义一些资产用来快速分配给其他策略c
type ServerFilterV1 struct {
	Name    []string `json:"name"`     // 名字完全匹配，支持*
	IpAddr  []string `json:"ip_addr"`  // IP 地址完全匹配，支持* 匹配所有
	EnvType []string `json:"env_type"` // 机器 Tags 中的 EnvType，支持* 匹配所有
	Team    []string `json:"team"`     // 机器 Tags 中的 Team，支持* 匹配所有
}

func (a ServerFilterV1) ToString() string {
	return fmt.Sprintf("%v", a)
}

func (a ServerFilterV1) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ServerFilterV1) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}
