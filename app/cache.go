package app

import (
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/patrickmn/go-cache"
	model "github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

func GetServers() model.Servers {
	log.Debugf("GetServers called")
	servers, found := App.Cache.Get("servers")
	if !found {
		return nil
	}
	return servers.(model.Servers)
}

func SetServers(servers model.Servers) {
	App.Cache.Set("servers", servers, cache.NoExpiration)
}

// func GetServerIDByIP(ip string) string {
// 	servers := GetServers()
// 	for _, server := range *servers {
// 		if server.Host == ip {
// 			return server.ID
// 		}
// 	}
// 	return ""
// }

func SetDBPolicyToCache() error {
	policies, err := App.JmsDBService.QueryAllPolicy()
	if err != nil {
		return err
	}
	App.Cache.Set("policies", policies, cache.NoExpiration)
	log.Debugf("set db policy to cache success")
	return nil
}

func GetUserPolicys(user model.User) []model.Policy {
	var matchPolicies []model.Policy
	if App.JmsDBService == nil {
		// 如果没有使用数据库，则默认都可见
		matchPolicies = append(matchPolicies, model.Policy{
			Actions:   model.All,
			IsEnabled: true,
			Users:     model.ArrayString{*user.Username},
		})
	} else {
		matchPolicies = QueryPolicyByUser(*user.Username)
	}
	return matchPolicies
}

func GetDBPolicy() []model.Policy {
	policies, found := App.Cache.Get("policies")
	if !found {
		return nil
	}
	return policies.([]model.Policy)
}

func QueryPolicyByUser(username string) []model.Policy {
	policies := GetDBPolicy()
	var matchPolicies []model.Policy
	// 精确返回
	for _, policy := range policies {
		if policy.Users.Contains(username) {
			if policy.IsDeleted || !policy.IsEnabled {
				continue
			}
			// log.Debugf("policy: %s", tea.Prettify(policy))
			matchPolicies = append(matchPolicies, policy)
		}
	}
	return matchPolicies
}

func GetUser(username string) (model.User, error) {
	if App.Config.WithDB.Enable {
		// 兼容数据库方式获取更多用户信息
		user, err := App.JmsDBService.DescribeUser(username)
		if err != nil {
			return user, err
		}
		return user, nil
	}
	return model.User{
		Username: tea.String(username),
	}, nil
}

func GetBroadcast() *model.Broadcast {
	if App.Config.WithDB.Enable {
		broadcast, err := App.JmsDBService.GetBroadcast()
		if err != nil {
			log.Errorf("GetBroadcast error: %s", err)
		}
		return broadcast
	}
	if App.Config.Broadcast == "" {
		return nil
	}
	return &model.Broadcast{
		Message: App.Config.Broadcast,
		Expires: time.Now().Add(24 * time.Hour), // 一直有效
	}
}
