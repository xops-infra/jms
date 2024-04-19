package instance

import (
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/db"
)

func LoadServer(conf *config.Config) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("LoadServer panic: %v", err)
		}
	}()

	keys := make(map[string]db.Key, 0)
	// 支持启用和不启用数据库获取 KEY 来源
	if app.App.Config.WithPolicy.Enable {
		// get keys from db
		resp, err := app.App.DBService.InternalLoad()
		if err != nil {
			log.Panicf("load keys failed: %v", err)
		}
		keys = resp
	} else {
		for id, keyName := range app.App.Config.Keys {
			keys[id] = db.Key{
				KeyName: keyName,
			}
		}
	}
	log.Debugf(tea.Prettify(keys))
	app.App.Cache.Set("keys", keys, 0)

	instanceAll := make(map[string]config.Server, 0)
	startTime := time.Now()
	instances := app.App.Server.QueryInstances(model.InstanceQueryInput{
		// Status: model.InstanceStatusRunning,
	})
	for _, instance := range instances {
		if instance.Status != model.InstanceStatusRunning {
			// for _, privateIp := range instance.PrivateIP {
			// 	if privateIp != nil && instance.Name != nil && instance.Status != nil {
			// 		log.Debugf("instance: %s ip:%s status: %s\n", *instance.Name, *privateIp, *instance.Status)
			// 	}
			// }
			continue
		}
		var keyName []*string
		for _, key := range instance.KeyName {
			if key == nil {
				break
			}
			// 解决 key大写不识别问题
			key = tea.String(strings.ToLower(*key))
			if _, ok := keys[*key]; ok {
				keyName = append(keyName, tea.String(keys[*key].KeyName))
			} else {
				log.Warnf("instance:%s key:%s not found in config.yml\n", *instance.Name, *key)
				continue
			}
		}
		if keyName == nil {
			// 只有被 jms配置纪录的 key才会被接管，否则会出现无法登录情况。
			continue
		}
		// log.Infof("instance:%s key: %s ips:%s\n", *instance.Name, *keyName, *instance.PrivateIP[0])
		if len(instance.PrivateIP) < 1 || instance.Tags == nil {
			continue
		}
		sshUser := fmtSuperUser(instance)
		instanceAll[*instance.PrivateIP[0]] = config.Server{
			ID:       *instance.InstanceID,
			Name:     tea.StringValue(instance.Name),
			Host:     *instance.PrivateIP[0],
			Port:     22,
			Profile:  instance.Profile,
			Region:   tea.StringValue(instance.Region),
			Status:   instance.Status,
			KeyPairs: keyName,
			SSHUsers: &sshUser,
			Tags:     *instance.Tags,
			Proxy:    fmtProxy(instance, conf),
		}
		log.Debugf(tea.Prettify(instanceAll[*instance.InstanceID].SSHUsers))
	}
	app.App.Cache.Set("servers", instanceAll, 0)
	log.Infof("%s len: %d", time.Since(startTime), len(instances))
}

func GetServers() map[string]config.Server {
	servers, found := app.App.Cache.Get("servers")
	if !found {
		return nil
	}
	return servers.(map[string]config.Server)
}

// 通过机器的密钥对 Key Name 获取对应的密钥Pem的路径
func getKeyPair(keyNames []*string) []db.Key {
	keysAll := make([]db.Key, 0)

	keys, ok := app.App.Cache.Get("keys")
	if !ok {
		return keysAll
	}
	configKeys := keys.(map[string]db.Key)

	for _, keyName := range keyNames {
		if keyName == nil {
			continue
		}
		lowKey := strings.ToLower(*keyName)
		if _, ok := configKeys[lowKey]; ok {
			keysAll = append(keysAll, configKeys[lowKey])
		}
	}
	return keysAll
}

// fmtSuperUser 支持多用户选择
func fmtSuperUser(instance *model.Instance) map[string]*config.SSHUser {
	keys := getKeyPair(instance.KeyName)
	sshUser := make(map[string]*config.SSHUser, 0)
	log.Debugf("platform: %s\n", *instance.Platform)
	for _, key := range keys {
		if strings.Contains(*instance.Platform, "Ubuntu") {
			sshUser["ubuntu"] = &config.SSHUser{
				SSHUsername:  "ubuntu",
				IdentityFile: key.KeyName,
				Base64Pem:    key.PemBase64,
			}
		} else if *instance.Platform == "Linux/UNIX" {
			sshUser["ec2-user"] = &config.SSHUser{
				SSHUsername:  "ec2-user",
				IdentityFile: key.KeyName,
				Base64Pem:    key.PemBase64,
			}
		} else {
			sshUser["root"] = &config.SSHUser{
				SSHUsername:  "root",
				IdentityFile: key.KeyName,
				Base64Pem:    key.PemBase64,
			}
		}
	}
	return sshUser
}

// fmtProxy
func fmtProxy(instance *model.Instance, conf *config.Config) *config.Proxy {
	// log.Debugf(tea.Prettify(instance), tea.Prettify(conf))
	for _, privateIp := range instance.PrivateIP {
		for _, proxy := range conf.Proxies {
			if strings.HasPrefix(*privateIp, proxy.IPPrefix) {
				log.Debugf(*privateIp, proxy.IPPrefix)
				return &config.Proxy{
					Host:     proxy.Host,
					Port:     proxy.Port,
					SSHUsers: proxy.SSHUsers,
				}
			}
		}
	}
	return nil
}
