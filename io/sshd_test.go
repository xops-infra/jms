package io

import (
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
	"github.com/xops-infra/jms/model"
	mcsModel "github.com/xops-infra/multi-cloud-sdk/pkg/model"
)

func TestMatchPolicyOwner(t *testing.T) {
	user := model.User{
		Username: tea.String("zhukun"),
	}
	server := model.Server{
		Tags: mcsModel.Tags{
			{
				Key:   "Owner",
				Value: "Zhukun",
			},
		},
	}
	assert.True(t, matchPolicyOwner(user, server))
}
