package pui_test

import (
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"

	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/core/pui"
)

func TestMatchServer(t *testing.T) {
	filter := db.ServerFilter{
		EnvType: tea.String("!prod"),
	}
	server := config.Server{
		Tags: model.Tags{
			{
				Key:   "EnvType",
				Value: "prod",
			},
		},
	}

	if pui.MatchServer(filter, server) {
		t.Error("should match")
	}

	server.Tags = model.Tags{
		{
			Key:   "EnvType",
			Value: "dev",
		},
	}
	if !pui.MatchServer(filter, server) {
		t.Error("should not match")
	}

}
