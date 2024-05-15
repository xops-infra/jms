package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
)

type Policy struct {
	ID           string        `json:"id" gorm:"column:id;primary_key;not null"`
	CreatedAt    *time.Time    `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    *time.Time    `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted    *bool         `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	Name         *string       `json:"name" gorm:"column:name;not null"`
	Users        ArrayString   `json:"users" gorm:"column:users;type:json;not null"`
	ServerFilter *ServerFilter `json:"server_filter" gorm:"column:server_filter;type:json;not null"`
	Actions      ArrayString   `json:"actions" gorm:"column:actions;type:json;not null"`
	ExpiresAt    *time.Time    `json:"expires_at" gorm:"column:expires_at;not null"`
	Approver     *string       `json:"approver" gorm:"column:approver"`       // 审批人
	ApprovalID   *string       `json:"approval_id" gorm:"column:approval_id"` // 审批ID
	IsEnabled    *bool         `json:"is_enabled" gorm:"column:is_enabled;default:false;not null"`
}

func (p *Policy) IsExpired() bool {
	return time.Since(*p.ExpiresAt) > 0
}

func (Policy) TableName() string {
	return "jms_go_policy"
}

type Action string
type Period string

const (
	Connect      Action = "connect"
	DenyConnect  Action = "deny_connect"
	Download     Action = "download"
	DenyDownload Action = "deny_download"
	Upload       Action = "upload"
	DenyUpload   Action = "deny_upload"

	OneDay   Period = "1d"
	OneWeek  Period = "1w"
	OneMonth Period = "1m"
	OneYear  Period = "1y"
	Forever  Period = "ever"
)

var (
	ConnectOnly        = ArrayString{Connect}
	DownloadOnly       = ArrayString{Download}
	UploadOnly         = ArrayString{Upload}
	ConnectAndDownload = ArrayString{Connect, Download}
	ConnectAndUpload   = ArrayString{Connect, Upload}
	DownloadAndUpload  = ArrayString{Download, Upload}
	All                = ArrayString{Connect, Download, Upload}

	DefaultPolicies = map[string]ArrayString{
		"All":                All,
		"ConnectOnly":        ConnectOnly,
		"DownloadOnly":       DownloadOnly,
		"UploadOnly":         UploadOnly,
		"ConnectAndDownload": ConnectAndDownload,
		"ConnectAndUpload":   ConnectAndUpload,
		"DownloadAndUpload":  DownloadAndUpload,
	}

	ExpireTimes = map[Period]time.Duration{
		OneDay:   time.Hour * 24,
		OneWeek:  time.Hour * 24 * 7,
		OneMonth: time.Hour * 24 * 30,
		OneYear:  time.Hour * 24 * 365,
		Forever:  time.Hour * 24 * 365 * 100,
	}
)

type PolicyQueryRequest struct {
	User *string `json:"user"`
}

type PolicyMut struct {
	Name         *string       `json:"name"`
	Users        ArrayString   `json:"users"`
	Groups       ArrayString   `json:"groups"`
	ServerFilter *ServerFilter `json:"server_filter"`
	Actions      ArrayString   `json:"actions"`
	ExpiresAt    *time.Time    `json:"expires_at"`
	IsEnabled    *bool         `json:"is_enabled"`
}

type ApprovalMut struct {
	Users        ArrayString   `json:"users" binding:"required"`
	Groups       ArrayString   `json:"groups"`
	Applicant    *string       `json:"applicant" binding:"required"` // 申请人AD名,或者email
	Name         *string       `json:"name"`
	Period       *Period       `json:"period"`  // 审批周期，默认一周
	Actions      []Action      `json:"actions"` // 申请动作，默认只有connect
	ServerFilter *ServerFilter `json:"server_filter" binding:"required"`
}

func (a *ApprovalMut) ToPolicyMut() *PolicyMut {
	defalutPeriod := time.Now().Add(ExpireTimes[OneWeek]) // 默认一周
	req := &PolicyMut{
		Name:         tea.String(fmt.Sprintf("%s-%s", *a.Applicant, time.Now().Format("20060102150405"))),
		Users:        a.Users,
		Groups:       a.Groups,
		ServerFilter: a.ServerFilter,
		ExpiresAt:    &defalutPeriod,
		Actions: ArrayString{
			Connect,
		},
	}
	if a.Name == nil {
		req.Name = a.Name
	}
	if a.Period != nil {
		expiresAt := time.Now().Add(ExpireTimes[*a.Period])
		req.ExpiresAt = &expiresAt
	}
	if len(a.Actions) > 0 {
		for _, action := range a.Actions {
			req.Actions = append(req.Actions, string(action))
		}
	}
	return req
}

type ApprovalResult struct {
	Applicant *string `json:"applicant"`
	IsPass    *bool   `json:"is_pass"`
}

// 支持!开头的反向匹配
// 默认没有匹配到标签的允许访问
func MatchServerByFilter(filter ServerFilter, server Server) bool {
	if filter.Name != nil {
		if *filter.Name == "*" || *filter.Name == server.Name {
			return true
		}
	}
	if filter.IpAddr != nil {
		if *filter.IpAddr == "*" || *filter.IpAddr == server.Host {
			return true
		}
	}
	if filter.EnvType != nil {
		if server.Tags.GetEnvType() == nil {
			return true
		}
		if strings.HasPrefix(*filter.EnvType, "!") {
			if strings.TrimPrefix(*filter.EnvType, "!") == *server.Tags.GetEnvType() {
				return false
			} else {
				return true
			}
		}
		if *filter.EnvType == "*" || *filter.EnvType == *server.Tags.GetEnvType() {
			return true
		}
	}
	if filter.Team != nil {
		if server.Tags.GetTeam() == nil {
			return true
		}
		if *filter.Team == "*" || *filter.Team == *server.Tags.GetTeam() {
			return true
		}
	}

	return false
}
