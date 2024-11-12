package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/noop/log"
)

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
	DenyALL            = ArrayString{string(DenyConnect), string(DenyDownload), string(DenyUpload)}
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
	Users ArrayString `json:"users" binding:"required"`
	// Groups       ArrayString     `json:"groups"`
	Applicant    *string         `json:"applicant" binding:"required"` // 申请人AD名,或者email
	Name         *string         `json:"name"`
	Period       *Period         `json:"period"`  // 审批周期，默认一周
	Actions      []Action        `json:"actions"` // 申请动作，默认只有connect
	ServerFilter *ServerFilterV1 `json:"server_filter" binding:"required"`
}

func (a *ApprovalMut) ToPolicyMut() *PolicyRequest {
	defalutPeriod := time.Now().Add(ExpireTimes[OneWeek]) // 默认一周
	req := &PolicyRequest{
		Name:           tea.String(fmt.Sprintf("%s-%s", *a.Applicant, time.Now().Format("20060102150405"))),
		Users:          a.Users,
		ServerFilterV1: a.ServerFilter,
		ExpiresAt:      &defalutPeriod,
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

/*
支持 ! 和 * 匹配；
为了支持 以下 4 个状态
正向匹配命中(后续直接返回 true)，
正向匹配没命中（后续处理下一个匹配），
反向匹配命中（后续直接 return false），
反向匹配没命中（后续处理下一个匹配）
*/
func stringMatch(std, judge string) bool {
	if std == "" && judge == "!*" {
		// 错误的输入直接返回false
		return false
	}

	negatedJudge := false
	if strings.HasPrefix(judge, "!") {
		judge = strings.TrimPrefix(judge, "!")
		negatedJudge = true
	}

	// 处理 * 开头的模糊匹配
	if judge == "*" {
		return !negatedJudge
	}

	// 处理包含*的模糊匹配情况
	if strings.Contains(judge, "*") {
		if strings.HasPrefix(std, strings.TrimSuffix(judge, "*")) {
			if !negatedJudge {
				return true
			}
		} else {
			if negatedJudge {
				return true
			}
		}
	} else {
		// 处理没有*的模糊匹配情况
		// log.Debugf("judge:%s and std:%s", judge, std)
		if std == judge {
			if !negatedJudge {
				return true
			}
		} else {
			// log.Debugf("not match negatedJudge: %s", negatedJudge)
			if negatedJudge {
				return true
			}
		}
	}
	return false
}

// 匹配服务器和过滤条件是否符合
// 支持多维度的并联匹配，ServerFilterV1如果属性没有为nil，则要进行联合匹配
func MatchServerByFilter(filter ServerFilterV1, server Server) bool {
	log.Debugf("filter:%s", tea.Prettify(filter))
	log.Debugf("server:%s", tea.Prettify(server))

	if filter.EnvType == nil && filter.Team == nil &&
		filter.Name == nil && filter.IpAddr == nil && filter.KV == nil {
		log.Errorf("filter is empty, return false")
		return false
	}

	IsMatchName := false
	if filter.Name != nil {
		for _, name := range filter.Name {
			if stringMatch(server.Name, name) {
				IsMatchName = true
				break
			}
		}
	} else {
		IsMatchName = true
	}

	IsMatchIP := false
	if filter.IpAddr != nil {
		for _, ip := range filter.IpAddr {
			if stringMatch(server.Host, ip) {
				IsMatchIP = true
				break
			}
		}
	} else {
		IsMatchIP = true
	}

	IsMatchEnvType := false
	if filter.EnvType != nil {
		if server.Tags.GetEnvType() != nil {
			for _, envType := range filter.EnvType {
				if stringMatch(*server.Tags.GetEnvType(), envType) {
					IsMatchEnvType = true
					break
				}
			}
		}
	} else {
		IsMatchEnvType = true
	}

	IsMatchTeam := false
	if filter.Team != nil {
		if server.Tags.GetTeam() != nil {
			for _, team := range filter.Team {
				if stringMatch(*server.Tags.GetTeam(), team) {
					IsMatchTeam = true
					break
				}
			}
		}
	} else {
		IsMatchTeam = true
	}

	// 判断自定义 KV 匹配
	IsMatchKV := false
	if filter.KV != nil {
		for _, tag := range server.Tags {
			if tag.Key == filter.KV.Key && tag.Value == filter.KV.Value {
				IsMatchKV = true
				break
			}
		}
	} else {
		IsMatchKV = true
	}

	if IsMatchName && IsMatchIP && IsMatchEnvType && IsMatchTeam && IsMatchKV {
		return true
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
	log.Debugf("systemPolicyCheck for user: %s", tea.Prettify(user))
	if systemPolicyCheck(user, server) {
		log.Debugf("system policy allow for user: %s", *user.Username)
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

		// 数据库查 policy的时候已经过滤了非当前用户的情况
		// if !dbPolicy.Users.Contains(*user.Username) {
		// 	log.Debugf("policy %s is not for user %s", dbPolicy.Name, *user.Username)
		// 	continue
		// }
		allow := policyCheck(inPutAction, server, dbPolicy)

		if allow == nil {
			continue
		}
		if !*allow {
			// 找到拒绝的策略直接拒绝
			log.Infof("deny policy got! %s '%s', stop check other policy", dbPolicy.ID, dbPolicy.Name)
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
	if policy.ServerFilterV1 == nil {
		log.Debugf("ServerFilterV1 is nil")
		return nil
	}
	if !MatchServerByFilter(*policy.ServerFilterV1, server) {
		// 不符合的机器直接跳过
		log.Debugf("server not match policy ignore %s", tea.Prettify(policy.ServerFilter))
		return nil
	}
	log.Debugf("server match policy allow for Policy %s", tea.Prettify(policy))
	// 符合的机器再判断 action
	for _, action := range policy.Actions {
		if string(inPutAction) == action {
			log.Debugf("action allow")
			return tea.Bool(true)
		}
		if string(inPutAction) == string(ReverseAction(Action(action))) {
			log.Debugf("got action deny")
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
