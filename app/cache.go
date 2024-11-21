package app

import (
	"time"

	model "github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

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
