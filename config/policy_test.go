package config_test

import (
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"

	. "github.com/xops-infra/jms/config"
)

func init() {
	// config.LoadYaml("/opt/jms/config.yaml")
	log.Default().WithLevel(log.DebugLevel).WithFilename("/tmp/test.log").Init()
}

func TestMatchServer(t *testing.T) {
	filter := ServerFilter{
		EnvType: tea.String("!prod"),
	}
	server := Server{
		Tags: model.Tags{
			{
				Key:   "EnvType",
				Value: "prod",
			},
		},
	}

	if MatchServerByFilter(filter, server) {
		t.Error("should match")
	}

	server.Tags = model.Tags{
		{
			Key:   "EnvType",
			Value: "dev",
		},
	}
	if !MatchServerByFilter(filter, server) {
		t.Error("should not match")
	}

}

// TEST MatchPolicy
func TestMatchPolicy(t *testing.T) {
	Conf.WithDB.Enable = true

	user := User{
		Username: tea.String("zhoushoujian"),
		Groups:   ArrayString{},
	}
	inPutAction := Connect
	server := Server{}
	policy := Policy{
		Name:      "test policy",
		IsEnabled: true,
		Users:     ArrayString{"zhoushoujian"},
		Actions:   ArrayString{"connect"},
		ExpiresAt: time.Now().Add(ExpireTimes[OneWeek]),
	}

	user.Groups = ArrayString{"admin"}
	{
		// 测试 admin 组
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
	}

	user.Groups = ArrayString{}
	{
		// 测试普通用户,IP 匹配
		policy.ServerFilter.IpAddr = tea.String("127.0.0.1")
		policy.ServerFilter.Name = nil
		policy.ServerFilter.EnvType = nil
		policy.ServerFilter.Team = nil
		server.Host = "127.0.0.1"
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
		server.Host = "89.0.142.86"
		assert.False(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
	}
	{
		// 普通用户，Name匹配
		policy.ServerFilter.Name = tea.String("test")
		policy.ServerFilter.EnvType = nil
		policy.ServerFilter.Team = nil
		policy.ServerFilter.IpAddr = nil
		server.Name = "test"
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
		server.Name = "test2"
		assert.False(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
	}
	{
		// 普通用户，EnvType匹配
		policy.ServerFilter.Team = nil
		policy.ServerFilter.Name = nil
		policy.ServerFilter.IpAddr = nil
		policy.ServerFilter.EnvType = tea.String("prod")
		server.Tags = model.Tags{
			{
				Key:   "EnvType",
				Value: "prod",
			},
		}
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
		server.Tags = model.Tags{
			{
				Key:   "EnvType",
				Value: "dev",
			},
		}
		assert.False(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
	}
	{
		// 普通用户，Team匹配
		policy.ServerFilter.Team = tea.String("ops")
		policy.ServerFilter.Name = nil
		policy.ServerFilter.EnvType = nil
		policy.ServerFilter.IpAddr = nil

		server.Tags = model.Tags{
			{
				Key:   "Team",
				Value: "ops",
			},
		}
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
		server.Tags = model.Tags{
			{
				Key:   "Team",
				Value: "others",
			},
		}
		assert.False(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
	}
	{
		// 普通用户，Owner匹配
		policy.ServerFilter.Team = nil
		policy.ServerFilter.Name = nil
		policy.ServerFilter.EnvType = nil
		policy.ServerFilter.IpAddr = nil

		server.Tags = model.Tags{
			{
				Key:   "Owner",
				Value: "zhoushoujian",
			},
		}
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
		server.Tags = model.Tags{
			{
				Key:   "Owner",
				Value: "xxx",
			},
		}
		assert.False(t, MatchPolicy(user, inPutAction, server, []Policy{
			policy,
		}))
	}

}

func TestMultipolicy(t *testing.T) {
	Conf.WithDB.Enable = true

	user := User{
		Username: tea.String("zhoushoujian"),
		Groups:   ArrayString{},
	}
	inPutAction := Connect

	defaultPolicy := Policy{
		IsEnabled: true,
		Users:     ArrayString{"zhoushoujian"},
		Actions:   ArrayString{"connect"},
		ExpiresAt: time.Now().Add(ExpireTimes[OneWeek]),
		ServerFilter: ServerFilter{
			IpAddr: tea.String("127.0.0.1"),
		},
	}
	{
		server := Server{
			Host: "127.0.0.1",
		}
		// 测试 deny 匹配
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			defaultPolicy,
		}))
		assert.False(t, MatchPolicy(user, inPutAction, server, []Policy{
			defaultPolicy,
			{
				IsEnabled: true,
				Users:     ArrayString{"zhoushoujian"},
				Actions:   ArrayString{string(DenyConnect)},
				ExpiresAt: time.Now().Add(ExpireTimes[OneWeek]),
				ServerFilter: ServerFilter{
					Name: tea.String("*"),
				},
			},
		}))

		// 测试 ! 匹配
		server.Tags = model.Tags{
			{
				Key:   "EnvType",
				Value: "prod",
			},
		}
		assert.False(t, MatchPolicy(user, inPutAction, server, []Policy{
			{
				IsEnabled: true,
				Users:     ArrayString{"zhoushoujian"},
				Actions:   ArrayString{string((Connect))},
				ExpiresAt: time.Now().Add(ExpireTimes[OneWeek]),
				ServerFilter: ServerFilter{
					EnvType: tea.String("!prod"),
				},
			},
		}))
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			{
				IsEnabled: true,
				Users:     ArrayString{"zhoushoujian"},
				Actions:   ArrayString{string((Connect))},
				ExpiresAt: time.Now().Add(ExpireTimes[OneWeek]),
				ServerFilter: ServerFilter{
					EnvType: tea.String("!dev"),
				},
			},
		}))

		// 测试 * 匹配
		server.Tags = model.Tags{
			{
				Key:   "Team",
				Value: "ops",
			},
		}
		assert.True(t, MatchPolicy(user, inPutAction, server, []Policy{
			{
				IsEnabled: true,
				Users:     ArrayString{"zhoushoujian"},
				Actions:   ArrayString{string((Connect))},
				ExpiresAt: time.Now().Add(ExpireTimes[OneWeek]),
				ServerFilter: ServerFilter{
					Team: tea.String("*"),
				},
			},
		}))
		assert.False(t, MatchPolicy(user, inPutAction, server, []Policy{
			{
				IsEnabled: true,
				Users:     ArrayString{"zhoushoujian"},
				Actions:   ArrayString{string((Connect))},
				ExpiresAt: time.Now().Add(ExpireTimes[OneWeek]),
				ServerFilter: ServerFilter{
					Team: tea.String("data"),
				},
			},
		}))
	}
}
