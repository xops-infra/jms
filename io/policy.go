package io

import (
	"fmt"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/model"

	"github.com/xops-infra/noop/log"
)

type PolicyIO struct {
	WithDB bool
}

func NewPolicy(withDB bool) *PolicyIO {
	return &PolicyIO{
		WithDB: withDB,
	}
}

// 对用户，策略，服务器，动作做权限判断
// onlyIp 用来兼容策略对上传下载的判断，因为上传下载信息只会有 IP 信息。
func (p *PolicyIO) MatchPolicy(user model.User, inPutAction model.Action, server model.Server, dbPolicies []model.Policy, onlyIp bool) bool {

	if !p.WithDB {
		// 没有启用数据库策略的直接通过
		log.Debugf("db is not enable, allow all")
		return true
	}
	log.Debugf("systemPolicyCheck for user: %s", tea.Prettify(user))
	if model.SystemPolicyCheck(user, server) {
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

// argsWithServer 是 root@10.9.x.x:/data/xx.zip 这一串组合字符，方法内会解析
func (p *PolicyIO) CheckPermission(argsWithServer string, user model.User, inputAction model.Action, matchPolicies []model.Policy) error {
	serverIP, err := model.ExtractIP(argsWithServer)
	if err != nil {
		return err
	}

	// 判断是否有权限
	if !p.MatchPolicy(user, inputAction, model.Server{
		Host: serverIP,
	}, matchPolicies, true) {
		return fmt.Errorf("user: %s has no permission to %s server: %s", *user.Username, inputAction, serverIP)
	}
	return nil
}
