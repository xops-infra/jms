package db

import (
	"fmt"
	"strings"

	"github.com/elfgzp/ssh"
	"github.com/google/uuid"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

// 支持单个用户多个公钥认证
func (d *DBService) AuthKey(username string, pub ssh.PublicKey) bool {
	keys, err := d.GetKeyByUsername(username)
	if err != nil {
		log.Errorf("get %s key error: %s", username, err)
		return false
	}
	for _, key := range keys {
		allowed, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(key.PublicKey))
		if ssh.KeysEqual(allowed, pub) {
			return true
		}
	}
	return false
}

func (d *DBService) GetKeyByUsername(username string) ([]model.AuthorizedKey, error) {
	var keys []model.AuthorizedKey
	err := d.DB.Where("user_name = ? and is_delete = false", username).Find(&keys).Error
	return keys, err
}

// addAuthorizedKey
func (d *DBService) AddAuthorizedKey(username string, pub string) error {
	// 先查询 pub key 是否已存在
	var count int64
	// 处理掉换行符
	pub = strings.ReplaceAll(pub, "\n", "")
	d.DB.Model(model.AuthorizedKey{}).Where("public_key = ?", pub).Count(&count)
	if count > 0 {
		return fmt.Errorf("key already exists")
	}
	key := &model.AuthorizedKey{
		IsDelete:  false,
		UUID:      uuid.NewString(),
		UserName:  username,
		PublicKey: pub,
	}
	return d.DB.Create(key).Error
}
