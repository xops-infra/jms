package config_test

import (
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"

	. "github.com/xops-infra/jms/config"
)

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
