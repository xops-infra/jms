package app

import (
	"github.com/patrickmn/go-cache"
	model "github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

func GetServers() model.Servers {
	servers, found := App.Cache.Get("servers")
	if !found {
		return nil
	}
	return servers.(model.Servers)
}

func SetServers(servers model.Servers) {
	App.Cache.Set("servers", servers, cache.NoExpiration)
}

func GetServerIDByIP(ip string) string {
	servers := GetServers()
	for _, server := range servers {
		if server.Host == ip {
			return server.ID
		}
	}
	return ""
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

func SetDBPolicyToCache() error {
	policies, err := App.JmsDBService.QueryAllPolicy()
	if err != nil {
		return err
	}
	App.Cache.Set("policies", policies, cache.NoExpiration)
	log.Infof("set db policy to cache success")
	return nil
}
