package instance

import (
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"
)

func LoadServer(conf *config.Config) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("LoadServer panic: %v", err)
		}
	}()

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
		var keyName *string
		for _, key := range instance.KeyName {
			if key == nil {
				break
			}
			// 解决 key大写不识别问题
			key = tea.String(strings.ToLower(*key))
			if _, ok := conf.Keys[*key]; ok {
				keyName = key
				break
			} else {
				log.Warnf("instance:%s key: %s not found in config.yml\n", *instance.Name, *key)
				continue
			}
		}
		if keyName == nil {
			// 只有被 jms配置纪录的 key才会被接管，否则会出现无法登录情况。
			continue
		}
		// log.Infof("instance:%s key: %s ips:%s\n", *instance.Name, *keyName, *instance.PrivateIP[0])
		sshUser := fmtSuperUser(instance)
		instanceAll[*instance.PrivateIP[0]] = config.Server{
			Name:     *instance.Name,
			Host:     *instance.PrivateIP[0],
			Port:     22,
			Profile:  instance.Profile,
			Region:   *instance.Region,
			Status:   instance.Status,
			KeyPair:  keyName,
			SSHUsers: &sshUser,
			Tags:     *instance.Tags,
			Proxy:    fmtProxy(instance, conf),
		}
		log.Debugf(tea.Prettify(instanceAll[*instance.InstanceID].SSHUsers))
	}
	_, found := app.App.Cache.Get("servers")
	if found {
		app.App.Cache.Delete("servers")
	}
	err := app.App.Cache.Add("servers", instanceAll, 0)
	if err != nil {
		log.Errorf("app.App.Cache.Add error: %s", err)
	}
	log.Infof("%s len: %d", time.Since(startTime), len(instances))
}

func GetServers() map[string]config.Server {
	servers, found := app.App.Cache.Get("servers")
	if !found {
		return nil
	}
	return servers.(map[string]config.Server)
}

// getKeyPair
func getKeyPair(keyNames []*string) *string {
	configKeys := app.App.Config.Keys

	for _, keyName := range keyNames {
		if keyName == nil {
			continue
		}
		lowKey := strings.ToLower(*keyName)
		if _, ok := configKeys[lowKey]; ok {
			return tea.String(configKeys[lowKey])
		}
	}
	return nil
}

// fmtSuperUser 支持多用户选择
func fmtSuperUser(instance *model.Instance) map[string]*config.SSHUser {
	keyPath := getKeyPair(instance.KeyName)
	sshUser := make(map[string]*config.SSHUser, 0)
	log.Debugf("platform: %s\n", *instance.Platform)
	if strings.Contains(*instance.Platform, "Ubuntu") {
		sshUser["ubuntu"] = &config.SSHUser{
			SSHUsername:  "ubuntu",
			IdentityFile: tea.StringValue(keyPath),
		}
	} else if *instance.Platform == "Linux/UNIX" {
		sshUser["ec2-user"] = &config.SSHUser{
			SSHUsername:  "ec2-user",
			IdentityFile: tea.StringValue(keyPath),
		}
	} else {
		sshUser["root"] = &config.SSHUser{
			SSHUsername:  "root",
			IdentityFile: tea.StringValue(keyPath),
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
