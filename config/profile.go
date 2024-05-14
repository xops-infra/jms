package config

import "gorm.io/gorm"

type CreateProfileRequest struct {
	Name    *string     `json:"name"`
	AK      *string     `json:"ak"`
	SK      *string     `json:"sk"`
	Cloud   *string     `json:"cloud"  default:"tencent"` // aws, aliyun, tencent
	Regions StringSlice `json:"regions"`
	Enabled bool        `json:"enabled" default:"true"` // 是否启用
}

type Profile struct {
	gorm.Model `json:"-"`
	UUID       string      `gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	Name       string      `gorm:"column:name;type:varchar(255);not null"`
	AK         string      `gorm:"column:ak;type:varchar(255);not null"`
	SK         string      `gorm:"column:sk;type:varchar(255);not null"` // 经过加密
	IsDelete   bool        `gorm:"column:is_delete;type:boolean;not null;default:false"`
	Cloud      string      `gorm:"column:cloud;type:varchar(255);not null"`
	Regions    StringSlice `gorm:"column:regions;type:json;not null"`
	Enabled    bool        `gorm:"column:enabled;type:boolean;not null;default:true"`
}

func (Profile) TableName() string {
	return "profile"
}
