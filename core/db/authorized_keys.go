package db

import (
	"fmt"
	"strings"

	"github.com/elfgzp/ssh"
	"github.com/google/uuid"
	"github.com/xops-infra/jms/model"
)

func (d *DBService) AuthKey(username string, pub ssh.PublicKey) bool {
	var key model.AuthorizedKey
	err := d.DB.Where("user_name = ? and is_delete = false", username).First(&key).Error
	if err != nil {
		return false
	}
	allowed, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(key.PublicKey))
	return ssh.KeysEqual(allowed, pub)
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
