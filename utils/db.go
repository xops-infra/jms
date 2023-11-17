package utils

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type Policy struct {
	Id           string       `json:"id" gorm:"column:id;primary_key;not null"`
	CreatedAt    *time.Time   `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    *time.Time   `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted    *bool        `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	Name         *string      `json:"name" gorm:"column:name;not null"`
	IsEnabled    *bool        `json:"is_enabled" gorm:"column:is_enabled;default:false;not null"`
	Users        ArrayString  `json:"users" gorm:"column:users;type:json;not null"`
	Groups       ArrayString  `json:"groups" gorm:"column:groups;type:json;not null"`
	ServerFilter ServerFilter `json:"server_filter" gorm:"column:server_filter;type:json;not null"`
	Actions      ArrayString  `json:"actions" gorm:"column:actions;type:json;not null"`
}

func (Policy) TableName() string {
	return "jms_go_policy"
}

type ArrayString []string

func (a ArrayString) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ArrayString) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}

// 可以预定义一些资产用来快速分配给其他策略c
type ServerFilter struct {
	Name    *string `json:"name" `
	IpAddr  *string `json:"ip_addr" `
	Owner   *string `json:"owner" `
	EnvType *string `json:"env_type"`
}

func (a ServerFilter) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ServerFilter) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}

type Action string

const (
	Login    Action = "login"
	Download Action = "download"
	Upload   Action = "upload"
)

func (a Action) String() string {
	return string(a)
}

func (a Action) TString() *Action {
	return &a
}

// initSQLite
func NewSQLite() *gorm.DB {
	return nil
}
