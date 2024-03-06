package policy

import (
	"fmt"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/utils"
)

type Policy struct {
	ID           string              `json:"id" gorm:"column:id;primary_key;not null"`
	CreatedAt    *time.Time          `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    *time.Time          `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted    *bool               `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	Name         *string             `json:"name" gorm:"column:name;not null"`
	Users        utils.ArrayString   `json:"users" gorm:"column:users;type:json;not null"`
	ServerFilter *utils.ServerFilter `json:"server_filter" gorm:"column:server_filter;type:json;not null"`
	Actions      utils.ArrayString   `json:"actions" gorm:"column:actions;type:json;not null"`
	ExpiresAt    *time.Time          `json:"expires_at" gorm:"column:expires_at;not null"`
	Approver     *string             `json:"approver" gorm:"column:approver"`       // 审批人
	ApprovalID   *string             `json:"approval_id" gorm:"column:approval_id"` // 审批ID
	IsEnabled    *bool               `json:"is_enabled" gorm:"column:is_enabled;default:false;not null"`
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
	Name         *string             `json:"name"`
	Users        utils.ArrayString   `json:"users"`
	Groups       utils.ArrayString   `json:"groups"`
	ServerFilter *utils.ServerFilter `json:"server_filter"`
	Actions      utils.ArrayString   `json:"actions"`
	ExpiresAt    *time.Time          `json:"expires_at"`
	IsEnabled    *bool               `json:"is_enabled"`
}

type ApprovalMut struct {
	Users        utils.ArrayString   `json:"users" binding:"required"`
	Groups       utils.ArrayString   `json:"groups"`
	Applicant    *string             `json:"applicant" binding:"required"` // 申请人AD名,或者email
	Name         *string             `json:"name"`
	Period       *Period             `json:"period"`  // 审批周期，默认一周
	Actions      []Action            `json:"actions"` // 申请动作，默认只有connect
	ServerFilter *utils.ServerFilter `json:"server_filter" binding:"required"`
}

func (a *ApprovalMut) ToPolicyMut() *PolicyMut {
	defalutPeriod := time.Now().Add(ExpireTimes[OneWeek]) // 默认一周
	req := &PolicyMut{
		Name:         tea.String(fmt.Sprintf("%s-%s", *a.Applicant, time.Now().Format("20060102150405"))),
		Users:        a.Users,
		Groups:       a.Groups,
		ServerFilter: a.ServerFilter,
		ExpiresAt:    &defalutPeriod,
		Actions: utils.ArrayString{
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
