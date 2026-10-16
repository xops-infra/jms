package io

import (
	"crypto/sha256"
	"fmt"

	"github.com/xops-infra/jms/model"
)

// AutomaticSSHUsers adds one automatic choice per credential; explicit choices remain available.
func AutomaticSSHUsers(users []model.SSHUser) []model.SSHUser {
	var result []model.SSHUser
	seen := make(map[string]bool)
	for _, user := range users {
		if user.Password != "" || (user.Base64Pem == "" && user.KeyName == "") {
			continue
		}
		identity := user.KeyName
		if user.Base64Pem != "" {
			identity = fmt.Sprintf("%x", sha256.Sum256([]byte(user.Base64Pem)))
		}
		if seen[identity] {
			continue
		}
		seen[identity] = true
		user.Auto = true
		result = append(result, user)
	}
	return result
}

// SelectSSHUser never changes an explicit username. Omitting it selects automatic authentication.
func SelectSSHUser(users []model.SSHUser, username, key string) (model.SSHUser, error) {
	var matches []model.SSHUser
	for _, user := range users {
		if (username != "" && user.UserName != username) || (key != "" && user.KeyName != key) {
			continue
		}
		matches = append(matches, user)
	}
	if len(matches) == 0 {
		return model.SSHUser{}, fmt.Errorf("ssh identity not found (user=%q key=%q)", username, key)
	}
	selected := matches[0]
	for _, user := range matches[1:] {
		sameKey := user.Base64Pem == selected.Base64Pem && (user.Base64Pem != "" || user.KeyName == selected.KeyName)
		if !sameKey || user.Password != selected.Password {
			return model.SSHUser{}, fmt.Errorf("ssh identity is ambiguous; specify a key")
		}
	}
	selected.Auto = username == ""
	return selected, nil
}
