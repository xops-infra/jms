package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/noop/log"
)

type PolicyRequest struct {
	Name         *string       `json:"name" binding:"required"`
	Users        ArrayString   `json:"users"`
	Actions      ArrayString   `json:"actions"`
	ServerFilter *ServerFilter `json:"server_filter" binding:"required"`
	ExpiresAt    *time.Time    `json:"expires_at"`
	IsEnabled    *bool         `json:"is_enabled"`
}

type Policy struct {
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

func (p *Policy) IsExpired() bool {
	return time.Since(p.ExpiresAt) > 0
}

func (Policy) TableName() string {
	return "jms_go_policy"
}

type Action string
type Period string

// 判断是否反向操作
func ReverseAction(action Action) Action {
	if action == Connect {
		return DenyConnect
	}
	if action == DenyConnect {
		return Connect
	}
	if action == Download {
		return DenyDownload
	}
	if action == DenyDownload {
		return Download
	}
	if action == Upload {
		return DenyUpload
	}
	if action == DenyUpload {
		return Upload
	}
	return action
}

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
	ConnectOnly        = ArrayString{string(Connect)}
	DownloadOnly       = ArrayString{string(Download)}
	UploadOnly         = ArrayString{string(Upload)}
	ConnectAndDownload = ArrayString{string(Connect), string(Download)}
	ConnectAndUpload   = ArrayString{string(Connect), string(Upload)}
	DownloadAndUpload  = ArrayString{string(Download), string(Upload)}
	All                = ArrayString{string(Connect), string(Download), string(Upload)}

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

type ApprovalMut struct {
	Users        ArrayString   `json:"users" binding:"required"`
	Groups       ArrayString   `json:"groups"`
	Applicant    *string       `json:"applicant" binding:"required"` // 申请人AD名,或者email
	Name         *string       `json:"name"`
	Period       *Period       `json:"period"`  // 审批周期，默认一周
	Actions      []Action      `json:"actions"` // 申请动作，默认只有connect
	ServerFilter *ServerFilter `json:"server_filter" binding:"required"`
}

func (a *ApprovalMut) ToPolicyMut() *PolicyRequest {
	defalutPeriod := time.Now().Add(ExpireTimes[OneWeek]) // 默认一周
	req := &PolicyRequest{
		Name:         tea.String(fmt.Sprintf("%s-%s", *a.Applicant, time.Now().Format("20060102150405"))),
		Users:        a.Users,
		ServerFilter: a.ServerFilter,
		ExpiresAt:    &defalutPeriod,
		Actions: ArrayString{
			string(Connect),
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

type MatchResult int

const (
	// 后续处理下一个匹配
	MatchContinue MatchResult = 0
	// 直接返回 true
	MatchTrue MatchResult = 1
	// 直接 return false
	MatchFalse MatchResult = 2
)

// 支持!开头的反向匹配
// 支持*开头的模糊匹配，192.168.*开头的匹配
// 支持!192.168.*开头的反向匹配
// 如果输入为“” 直接返回false
// 如果 judge 为!* 直接返回false
/*
为了支持 以下 4 个状态
正向匹配命中(后续直接返回 true)，正向匹配没命中（后续处理下一个匹配），反向匹配命中（后续直接 return false），反向匹配没命中（后续处理下一个匹配）
return 返回状态 0（后续处理下一个匹配）,1(后续直接返回 true),2后续直接 return false）
*/
func stringMatch(std, judge string) MatchResult {
	if std == "" && judge == "!*" {
		return MatchContinue
	}

	negatedJudge := false
	if strings.HasPrefix(judge, "!") {
		judge = strings.TrimPrefix(judge, "!")
		negatedJudge = true
	}

	// 处理 * 开头的模糊匹配
	if judge == "*" {
		return MatchTrue
	}

	// 处理包含*的模糊匹配情况
	if strings.Contains(judge, "*") {
		if strings.HasPrefix(std, strings.TrimSuffix(judge, "*")) {
			if negatedJudge {
				return MatchFalse
			} else {
				return MatchTrue
			}
		}
	} else {
		log.Debugf("judge:%s and std:%s", judge, std)
		if negatedJudge && std != judge {
			// 取反未命中时候直接 return MatchTrue
			return MatchTrue
		} else if !negatedJudge && std == judge {
			// 没取反命中时候直接 return MatchTrue
			return MatchTrue
		} else if negatedJudge && std == judge {
			// 取反命中时候直接 return MatchFalse
			return MatchFalse
		}
	}
	return MatchContinue
}

// 匹配服务器和过滤条件是否符合
func MatchServerByFilter(filter ServerFilter, server Server) bool {
	log.Debugf("filter:%s", tea.Prettify(filter))
	log.Debugf("server:%s", tea.Prettify(server))
	for _, name := range filter.Name {
		switch stringMatch(server.Name, name) {
		case MatchTrue:
			return true
		case MatchFalse:
			return false
		default:
			continue
		}
	}
	for _, ip := range filter.IpAddr {
		switch stringMatch(server.Host, ip) {
		case MatchTrue:
			return true
		case MatchFalse:
			return false
		default:
			continue
		}
	}
	if server.Tags.GetEnvType() != nil {
		for _, envType := range filter.EnvType {
			switch stringMatch(*server.Tags.GetEnvType(), envType) {
			case MatchTrue:
				return true
			case MatchFalse:
				return false
			default:
				continue
			}
		}
	}
	if server.Tags.GetTeam() != nil {
		for _, team := range filter.Team {
			switch stringMatch(*server.Tags.GetTeam(), team) {
			case MatchTrue:
				return true
			case MatchFalse:
				return false
			default:
				continue
			}
		}
	}

	return false
}

// Owner和用户一样则有权限
func MatchPolicyOwner(user User, server Server) bool {
	if server.Tags.GetOwner() != nil && *server.Tags.GetOwner() == *user.Username {
		return true
	}
	return false
}

// 用户组一致则有权限
// admin有所有权限
func MatchUserGroup(user User, server Server) bool {
	if user.Groups != nil {
		if user.Groups.Contains("admin") {
			return true
		}
		if server.Tags.GetTeam() != nil {
			for _, group := range user.Groups {
				if *server.Tags.GetTeam() == group {
					return true
				}
			}
		} else {
			return false
		}

	}
	return false
}

// 对用户，策略，服务器，动作做权限判断
func MatchPolicy(user User, inPutAction Action, server Server, dbPolicies []Policy) bool {

	if !Conf.WithDB.Enable {
		// 没有启用数据库策略的直接通过
		log.Debugf("db is not enable, allow all")
		return true
	}

	if systemPolicyCheck(user, server) {
		return true
	}

	isOK := false
	for _, dbPolicy := range dbPolicies {
		if !dbPolicy.IsEnabled {
			log.Debugf("policy %s is disabled", dbPolicy.Name)
			continue
		}
		// 策略失效也直接 pass
		if dbPolicy.ExpiresAt.Before(time.Now()) {
			log.Debugf("policy %s is expired", dbPolicy.Name)
			continue
		}

		if !dbPolicy.Users.Contains(*user.Username) {
			log.Debugf("policy %s is not for user %s", dbPolicy.Name, *user.Username)
			continue
		}
		allow := policyCheck(inPutAction, server, dbPolicy)

		if allow == nil {
			continue
		}
		if !*allow {
			// 找到拒绝的策略直接拒绝
			return false
		}
		// 找到允许的策略继续多策略校验
		isOK = true
	}
	return isOK
}

// System level
func systemPolicyCheck(user User, server Server) bool {
	if user.Groups.Contains("admin") {
		log.Debugf("admin allow")
		return true
	}
	// 用户组一致则有权限
	if MatchUserGroup(user, server) {
		log.Debugf("team allow")
		return true
	}
	// Owner和用户一样则有权限
	if MatchPolicyOwner(user, server) {
		log.Debugf("owner allow")
		return true
	}
	return false
}

// Admin level check, only find ok, default deny
func policyCheck(inPutAction Action, server Server, policy Policy) *bool {
	if !MatchServerByFilter(policy.ServerFilter, server) {
		// 不符合的机器直接跳过
		log.Debugf("server not match policy ignore")
		return nil
	}
	// 符合的机器再判断 action
	for _, action := range policy.Actions {
		if string(inPutAction) == action {
			log.Debugf("action allow")
			return tea.Bool(true)
		}
		if string(inPutAction) == string(ReverseAction(Action(action))) {
			log.Debugf("action deny")
			return tea.Bool(false)
		}
	}
	return nil
}

// 审批表单目前只支持prod,dev,stage,none
// todo:判断策略属于审批的那个单子
func FmtDingtalkApproveFile(envType []string) string {
	return "prod"
}
