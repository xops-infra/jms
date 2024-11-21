package io

import (
	"sync"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/core/db"
	. "github.com/xops-infra/jms/model"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"
)

type InstanceIO struct {
	mcsS         model.CommonContract
	db           *db.DBService
	localServers []LocalServer
}

func NewInstance(m model.CommonContract, db *db.DBService, localServers []LocalServer) *InstanceIO {
	return &InstanceIO{
		mcsS: m,
		db:   db,

		localServers: localServers,
	}
}

func (i *InstanceIO) LoadServer() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("LoadServer panic: %v", err)
		}
	}()

	// 获取最新的 Profile信息
	profiles, err := i.db.LoadProfile()
	if err != nil {
		log.Errorf("LoadProfile error: %v", err)
		return
	}

	mcsServers := make(map[string]model.Instance, 2000)
	startTime := time.Now()
	wg := sync.WaitGroup{}
	for _, profile := range profiles {
		log.Debugf(tea.Prettify(profile))
		log.Debugf("profile: %s is enabled: %t", *profile.Name, profile.Enabled)
		if !profile.Enabled {
			continue
		}
		wg.Add(1)
		go func(profile CreateProfileRequest) {
			for _, region := range profile.Regions {
				log.Debugf("get instances profile: %s region: %s", *profile.Name, region)
				input := model.InstanceFilter{}
				for {
					resps, err := i.mcsS.DescribeInstances(*profile.Name, region, input)
					if err != nil {
						log.Errorf("%s %s DescribeInstances error: %v", *profile.Name, region, err)
						break
					}
					for _, instance := range resps.Instances {
						if instance.InstanceID == nil {
							log.Errorf("instance id is nil, %s", tea.Prettify(instance))
							continue
						}
						if _, ok := mcsServers[*instance.InstanceID]; ok {
							log.Warnf("instance %s already exist", *instance.InstanceID)
							continue
						}
						mcsServers[*instance.InstanceID] = instance
					}
					if resps.NextMarker == nil {
						// log.Warnf("break get instances profile: %s region: %s len: %d", *profile.Name, region, len(mcsServers))
						break
					}
					input.NextMarker = resps.NextMarker
				}
			}
			wg.Done()
			log.Infof("get instances profile: %s len: %d", *profile.Name, len(mcsServers))
		}(profile)
	}
	wg.Wait()
	// 入库
	err = i.db.UpdateServerWithDelete(fmtServer(i.localServers, mcsServers))
	if err != nil {
		log.Errorf("update server error: %v", err)
		return
	}
	log.Infof("load server finished cost: %s ", time.Since(startTime))
}

func (i *InstanceIO) GetServerCount() int {
	servers, err := i.db.LoadServer()
	if err != nil {
		return 0
	}
	return len(servers)
}

func fmtServer(localServers []LocalServer, instances map[string]model.Instance) Servers {
	var instanceAll Servers
	for _, instance := range instances {
		if instance.Status != model.InstanceStatusRunning {
			continue
		}

		// 支持一个机器多个 key
		var keyName []string
		if instance.KeyIDs == nil {
			log.Warnf("instance:%s key is nil", *instance.Name)
		} else {
			for _, key := range instance.KeyIDs {
				if key == nil {
					continue
				}
				keyName = append(keyName, *key)
			}
		}

		if len(instance.PrivateIP) < 1 {
			log.Errorf("instance: %s private ip is empty", *instance.Name)
			continue
		}
		newInstance := Server{
			ID:       *instance.InstanceID,
			Name:     tea.StringValue(instance.Name),
			Host:     *instance.PrivateIP[0],
			Port:     22,
			Profile:  instance.Profile,
			Region:   tea.StringValue(instance.Region),
			Status:   instance.Status,
			KeyPairs: keyName,
			Tags:     *instance.Tags,
		}
		instanceAll = append(instanceAll, newInstance)
	}

	// 载入自己配置服务器
	for _, server := range localServers {
		instanceAll = append(instanceAll, Server{
			ID:     "local_config",
			Name:   server.Name,
			Host:   server.Host,
			Port:   server.Port,
			Status: model.InstanceStatusRunning, // 配置加入的默认为running
		})
	}

	// instanceAll.SortByName()
	return instanceAll
}
