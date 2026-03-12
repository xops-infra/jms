package io

import (
	"fmt"
	"strings"

	"github.com/xops-infra/jms/model"
)

func summarizeKey(key model.AddKeyRequest) string {
	return fmt.Sprintf("key_name=%s key_id=%s user=%s profile=%s",
		safeString(key.IdentityFile),
		safeString(key.KeyID),
		safeString(key.UserName),
		safeString(key.Profile),
	)
}

func summarizeSSHUsers(users []model.SSHUser) string {
	if len(users) == 0 {
		return "[]"
	}
	items := make([]string, 0, len(users))
	for _, u := range users {
		items = append(items, fmt.Sprintf("{user=%s key=%s}", u.UserName, safeValue(u.KeyName)))
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func safeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func safeValue(value string) string {
	if value == "" {
		return ""
	}
	return value
}
