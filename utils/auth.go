package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/elfgzp/ssh"
	"github.com/xops-infra/noop/log"
)

func AuthFromFile(ctx ssh.Context, key ssh.PublicKey, sshDir string) bool {
	hostAuthorizedKeys := sshDir + "authorized_keys"
	data, err := os.ReadFile(hostAuthorizedKeys)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, ctx.User()) {
			allowed, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(line))
			if ssh.KeysEqual(key, allowed) {
				log.Debugf("user: %s, pub: %s", ctx.User(), line)
				return true
			}
		}
	}
	return false
}

func AddAuthToFile(user, pubKey, sshDir string) error {
	hostAuthorizedKeys := sshDir + "authorized_keys"
	data, err := os.ReadFile(hostAuthorizedKeys)
	if err != nil {
		log.Error(err.Error())
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, pubKey) {
			return fmt.Errorf("key already exists")
		}
	}
	// 将公钥添加到authorized_keys的第一行
	f, err := os.OpenFile(hostAuthorizedKeys, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(user + " " + pubKey + "\n" + string(data))
	if err != nil {
		return err
	}
	log.Infof("add pub key: %s to %s success", pubKey, hostAuthorizedKeys)
	return nil
}
