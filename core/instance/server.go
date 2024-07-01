package instance

import (
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/patrickmn/go-cache"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/model"
)

// 支持数据库和配置文件两种方式获取 KEY以及 Profile.
func LoadServer(conf *Config) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("LoadServer panic: %v", err)
		}
	}()

	var mcsServers []model.Instance
	startTime := time.Now()
	for _, profile := range conf.Profiles {
		log.Debugf(tea.Prettify(profile))
		log.Debugf("profile: %s is enabled: %t", *profile.Name, profile.Enabled)
		if !profile.Enabled {
			continue
		}
		for _, region := range profile.Regions {
			log.Infof("get instances profile: %s region: %s", *profile.Name, region)
			input := model.InstanceFilter{}
			for {
				resps, err := app.App.McsServer.DescribeInstances(*profile.Name, region, input)
				if err != nil {
					log.Errorf("%s %s DescribeInstances error: %v", *profile.Name, region, err)
					break
				}
				mcsServers = append(mcsServers, resps.Instances...)
				if resps.NextMarker == nil {
					break
				}
				input.NextMarker = resps.NextMarker
			}
		}
	}
	instanceAll := fmtServer(mcsServers, conf.Keys.ToMapWithID())
	app.App.Cache.Set("servers", instanceAll, cache.NoExpiration)
	log.Infof("%s len: %d", time.Since(startTime), len(instanceAll))
}

func fmtServer(instances []model.Instance, keys map[string]AddKeyRequest) Servers {
	var instanceAll Servers
	for _, instance := range instances {
		if instance.Status != model.InstanceStatusRunning {
			continue
		}
		var keyName []*string
		for _, key := range instance.KeyIDs {
			if key == nil {
				break
			}
			// 解决 key大写不识别问题
			key = tea.String(strings.ToLower(*key))
			if _, ok := keys[*key]; ok {
				keyName = append(keyName, keys[*key].IdentityFile)
			} else {
				log.Infof("instance: %s key: %s not found in config", *instance.Name, *key)
				continue
			}
		}
		if keyName == nil {
			// 只有被 jms配置纪录的 key才会被接管，否则会出现无法登录情况。
			continue
		}
		// log.Infof("instance:%s key: %s ips:%s\n", *instance.Name, *keyName, *instance.PrivateIP[0])
		if len(instance.PrivateIP) < 1 {
			log.Errorf("instance: %s private ip is empty", *instance.Name)
			continue
		}
		sshUser := fmtSuperUser(instance)
		newInstance := Server{
			ID:       *instance.InstanceID,
			Name:     tea.StringValue(instance.Name),
			Host:     *instance.PrivateIP[0],
			Port:     22,
			Profile:  instance.Profile,
			Region:   tea.StringValue(instance.Region),
			Status:   instance.Status,
			KeyPairs: keyName,
			SSHUsers: sshUser,
			Tags:     *instance.Tags,
		}
		instanceAll = append(instanceAll, newInstance)
	}
	// 载入自己配置服务器
	for _, server := range app.App.Config.LocalServers {
		instanceAll = append(instanceAll, Server{
			ID:     "local_config",
			Name:   server.Name,
			Host:   server.Host,
			Port:   server.Port,
			Status: model.InstanceStatusRunning, // 配置加入的默认为running
			SSHUsers: []SSHUser{
				{
					UserName: server.User,
					Password: server.Passwd,
				},
			},
		})
	}

	instanceAll.SortByName()
	return instanceAll
}

func GetServers() Servers {
	servers, found := app.App.Cache.Get("servers")
	if !found {
		return nil
	}
	return servers.(Servers)
}

// 通过机器的密钥对 KeyIDs 获取对应的密钥Pem的路径
func getKeyPair(keyIDS []*string) []AddKeyRequest {
	keysAll := make([]AddKeyRequest, 0)
	configKeys := app.App.Config.Keys.ToMapWithID()
	for _, keyID := range keyIDS {
		if keyID == nil {
			continue
		}
		if key, ok := configKeys[*keyID]; ok {
			keysAll = append(keysAll, key)
		}
	}
	return keysAll
}

// fmtSuperUser 支持多用户选择
func fmtSuperUser(instance model.Instance) []SSHUser {
	keys := getKeyPair(instance.KeyIDs)
	var sshUser []SSHUser
	for _, key := range keys {
		u := SSHUser{}
		// KeyName 是支持本地读取内容的
		if key.IdentityFile != nil {
			u.KeyName = tea.StringValue(key.IdentityFile)
		}
		if key.PemBase64 != nil {
			u.Base64Pem = tea.StringValue(key.PemBase64)
		}

		if strings.Contains(*instance.Platform, "Ubuntu") {
			u.UserName = "ubuntu"
		} else if *instance.Platform == "Linux/UNIX" {
			u.UserName = "ec2-user"
		} else {
			u.UserName = "root"
		}
		sshUser = append(sshUser, u)
	}
	return sshUser
}
