package config

import "gorm.io/gorm"

type AuthorizedKey struct {
	gorm.Model
	IsDelete  bool   `gorm:"column:is_delete;type:boolean;not null;default:false"`
	UUID      string `gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	UserName  string `gorm:"column:user_name;type:varchar(255);not null"` // ad用户名
	PublicKey string `gorm:"column:public_key;type:text;not null"`
}

// table name
func (AuthorizedKey) TableName() string {
	return "authorized_keys"
}
