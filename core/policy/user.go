package policy

import (
	"time"

	"github.com/xops-infra/jms/utils"
)

type User struct {
	Id        string            `json:"id" gorm:"column:id;primary_key;not null"`
	CreatedAt *time.Time        `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time        `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted *bool             `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	Username  *string           `json:"username" gorm:"column:username;not null"`
	Passwd    []byte            `json:"passwd" gorm:"column:passwd"` // 密码计算为md5保存
	Email     *string           `json:"email" gorm:"column:email"`
	Groups    utils.ArrayString `json:"groups" gorm:"column:groups;type:json"`
	IsLdap    *bool             `json:"is_ldap" gorm:"column:is_ldap;default:false;not null"`
}

func (User) TableName() string {
	return "jms_go_users"
}

type CreateUserRequest struct {
	Name   *string           `json:"name" binding:"required"`
	Email  *string           `json:"email" binding:"required"`
	Groups utils.ArrayString `json:"groups" binding:"required"`
	Passwd *string           `json:"passwd" binding:"required"`
	IsLdap *bool             `json:"is_ldap"`
}

type Group struct {
	Id        string     `json:"id" gorm:"column:id;primary_key;not null"`
	CreatedAt *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted *bool      `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	Name      *string    `json:"name" gorm:"column:name;not null"`
}

func (Group) TableName() string {
	return "jms_go_groups"
}
