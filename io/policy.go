package io

import (
	"fmt"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/model"

	"github.com/xops-infra/noop/log"
)

// 对用户，策略，服务器，动作做权限判断
// onlyIp 用来兼容策略对上传下载的判断，因为上传下载信息只会有 IP 信息。
func (p *SshdIO) MatchPolicy(user model.User, inPutAction model.Action, server model.Server, dbPolicies []model.Policy, onlyIp bool) bool {

	if p.db == nil {
		// 没有启用数据库策略的直接通过
		log.Debugf("db is not enable, allow all")
		return true
	}
	if p.SystemPolicyCheck(user, server) {
		log.Debugf("system policy allow for user: %s", tea.Prettify(user))
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
		allow := model.PolicyCheck(inPutAction, server, dbPolicy, onlyIp)

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

func (p *SshdIO) GetUserPolicys(username string) []model.Policy {
	var matchPolicies []model.Policy
	if p.db == nil {
		// 如果没有使用数据库，则默认都可见
		matchPolicies = append(matchPolicies, model.Policy{
			Actions:   model.All,
			IsEnabled: true,
			Users:     model.ArrayString{username},
		})
	} else {
		policies, err := p.db.QueryAllPolicy()
		if err != nil {
			log.Errorf("query all policy error: %s", err)
			return nil
		}
		for _, policy := range policies {
			if policy.Users.Contains(username) {
				if policy.IsDeleted || !policy.IsEnabled {
					continue
				}
				// log.Debugf("policy: %s", tea.Prettify(policy))
				matchPolicies = append(matchPolicies, policy)
			}
		}
	}
	return matchPolicies
}

// argsWithServer 是 root@10.9.x.x:/data/xx.zip 这一串组合字符，方法内会解析
func (p *SshdIO) CheckPermission(argsWithServer string, user model.User, inputAction model.Action) error {
	serverIP, err := model.ExtractIP(argsWithServer)
	if err != nil {
		return err
	}
	log.Debugf("serverIP: %s", serverIP)
	// 丰富 server
	server, err := p.db.GetInstanceByHost(serverIP)
	if err != nil {
		return fmt.Errorf("can not find server %s in jms try again later", serverIP)
	}
	dbPolicies := p.GetUserPolicys(*user.Username)
	// 判断是否有权限
	if !p.MatchPolicy(user, inputAction, *server, dbPolicies, true) {
		return fmt.Errorf("user: %s has no permission to %s server: %s", *user.Username, inputAction, serverIP)
	}
	return nil
}

// System level
func (p *SshdIO) SystemPolicyCheck(user model.User, server model.Server) bool {

	if user.Groups.Contains("admin") {
		log.Debugf("admin allow")
		return true
	}
	// 用户组一致则有权限
	if matchUserGroup(user, server) {
		log.Debugf("team allow")
		return true
	}
	// Owner和用户一样则有权限
	if matchPolicyOwner(user, server) {
		log.Debugf("owner allow")
		return true
	}
	return false
}

// 用户组一致则有权限
// admin有所有权限
func matchUserGroup(user model.User, server model.Server) bool {
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

// Owner和用户一样则有权限
func matchPolicyOwner(user model.User, server model.Server) bool {
	if server.Tags.GetOwner() != nil && *server.Tags.GetOwner() == *user.Username {
		return true
	}
	return false
}
