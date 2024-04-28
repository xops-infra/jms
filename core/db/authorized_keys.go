package db

import (
	"fmt"
	"strings"

	"github.com/elfgzp/ssh"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthorizedKey struct {
	gorm.Model
	IsDelete  bool   `gorm:"column:is_delete;type:boolean;not null;default:false"`
	UUID      string `gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	UserName  string `gorm:"column:user_name;type:varchar(255);not null"` // ad用户名
	PublicKey string `gorm:"column:public_key;type:text;not null"`
}

// table name
func (d *DBService) TableName() string {
	return "authorized_keys"
}

func (d *DBService) AuthKey(username string, pub ssh.PublicKey) bool {
	var key AuthorizedKey
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
	d.DB.Model(AuthorizedKey{}).Where("public_key = ?", pub).Count(&count)
	if count > 0 {
		return fmt.Errorf("key already exists")
	}
	key := &AuthorizedKey{
		IsDelete:  false,
		UUID:      uuid.NewString(),
		UserName:  username,
		PublicKey: pub,
	}
	return d.DB.Create(key).Error
}
