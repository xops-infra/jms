package db

import "gorm.io/gorm"

type KeyTable struct {
	gorm.Model
	IsDelete bool   `gorm:"type:boolean;not null;default:false"`
	UUID     string `gorm:"type:varchar(36);unique_index;not null"`
	KeyID    string `gorm:"type:varchar(36);unique_index;not null"`
	KeyFile  string `gorm:"type:varchar(255);not null"`
	PemMd5   string `gorm:"type:text;not null"`
}

func (KeyTable) TableName() string {
	return "key_table"
}

type DB interface {
	ListKey() ([]KeyTable, error)
	CreateKey(keyID, keyName string, PemMd5 string) error
	DeleteKey(keyID string) error
}
