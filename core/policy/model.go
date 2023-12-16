package policy

import (
	"time"

	"github.com/xops-infra/jms/utils"
)

type Policy struct {
	Id           string              `json:"id" gorm:"column:id;primary_key;not null"`
	CreatedAt    *time.Time          `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    *time.Time          `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted    *bool               `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	Name         *string             `json:"name" gorm:"column:name;not null"`
	Users        utils.ArrayString   `json:"users" gorm:"column:users;type:json;not null"`
	Groups       utils.ArrayString   `json:"groups" gorm:"column:groups;type:json;not null"`
	ServerFilter *utils.ServerFilter `json:"server_filter" gorm:"column:server_filter;type:json;not null"`
	Actions      utils.ArrayString   `json:"actions" gorm:"column:actions;type:json;not null"`
	ExpiresAt    *time.Time          `json:"expires_at" gorm:"column:expires_at;not null"`
	Approver     *string             `json:"approver" gorm:"column:approver"` // 审批人
	IsEnabled    *bool               `json:"is_enabled" gorm:"column:is_enabled;default:false;not null"`
}

func (p *Policy) IsExpired() bool {
	return time.Since(*p.ExpiresAt) > 0
}

func (Policy) TableName() string {
	return "jms_go_policy"
}

type Action string

const (
	Connect      Action = "connect"
	DenyConnect  Action = "deny_connect"
	Download     Action = "download"
	DenyDownload Action = "deny_download"
	Upload       Action = "upload"
	DenyUpload   Action = "deny_upload"
)

var (
	ConnectOnly        = utils.ArrayString{Connect}
	DownloadOnly       = utils.ArrayString{Download}
	UploadOnly         = utils.ArrayString{Upload}
	ConnectAndDownload = utils.ArrayString{Connect, Download}
	ConnectAndUpload   = utils.ArrayString{Connect, Upload}
	DownloadAndUpload  = utils.ArrayString{Download, Upload}
	All                = utils.ArrayString{Connect, Download, Upload}

	DefaultPolicies = map[string]utils.ArrayString{
		"All":                All,
		"ConnectOnly":        ConnectOnly,
		"DownloadOnly":       DownloadOnly,
		"UploadOnly":         UploadOnly,
		"ConnectAndDownload": ConnectAndDownload,
		"ConnectAndUpload":   ConnectAndUpload,
		"DownloadAndUpload":  DownloadAndUpload,
	}

	ExpireTimes = map[string]time.Duration{
		"1d":   time.Hour * 24,
		"1w":   time.Hour * 24 * 7,
		"1m":   time.Hour * 24 * 30,
		"1y":   time.Hour * 24 * 365,
		"ever": time.Hour * 24 * 365 * 100,
	}
)

type PolicyQueryRequest struct {
	User *string `json:"user"`
}

type CreatePolicyRequest struct {
	Name         *string             `json:"name" binding:"required"`
	Users        utils.ArrayString   `json:"users"`
	Groups       utils.ArrayString   `json:"groups"`
	ServerFilter *utils.ServerFilter `json:"server_filter" binding:"required"`
	Actions      utils.ArrayString   `json:"actions" binding:"required"`
	ExpiresAt    *time.Time          `json:"expires_at" binding:"required"`
}
