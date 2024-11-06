package db

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	. "github.com/xops-infra/jms/model"
	"gorm.io/gorm"
)

type DBService struct {
	DB *gorm.DB
}

func NewJmsDbService(db *gorm.DB) *DBService {
	return &DBService{
		DB: db,
	}
}

func (d *DBService) NeedApprove(username string) ([]*Policy, error) {
	// 是否 admin组，且有需要审批的策略
	var policies []*Policy
	user, err := d.DescribeUser(username)
	if err != nil {
		return nil, err
	}
	if user.Groups == nil {
		return nil, nil
	}
	if user.Groups.Contains("admin") {
		if err := d.DB.Where("is_enabled = ?", false).Where("approver is null").Find(&policies).Error; err != nil {
			return nil, err
		}
	}
	return policies, nil
}

func (d *DBService) DescribeUser(name string) (User, error) {
	var user User
	if strings.Contains(name, "@") {
		if err := d.DB.Where("email = ?", name).First(&user).Error; err != nil {
			return user, err
		}
		return user, nil
	}
	if err := d.DB.Where("username = ?", name).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func (d *DBService) QueryUserByGroup(group string) ([]User, error) {
	var users []User
	// json 字段不支持like查询
	if err := d.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	// 提高准确度
	var matchUsers []User
	for _, user := range users {
		if user.Groups.Contains(group) {
			matchUsers = append(matchUsers, user)
		}
	}
	return matchUsers, nil
}

func (d *DBService) QueryAllUser() ([]User, error) {
	var users []User
	if err := d.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// 自带校验是否存在
func (d *DBService) CreateUser(req *UserRequest) (string, error) {
	user := &User{
		Username:       req.Username,
		Email:          req.Email,
		Groups:         req.Groups,
		DingtalkID:     req.DingtalkID,
		DingtalkDeptID: req.DingtalkDeptID,
	}
	if req.Passwd != nil {
		// base64加密
		user.Passwd = tea.String(base64.StdEncoding.EncodeToString([]byte(*req.Passwd)))
	}
	// 判断用户是否存在
	var count int64
	if err := d.DB.Model(&User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("user already exists")
	}

	user.ID = uuid.NewString()
	if d.DB.Create(user).Error != nil {
		return "", d.DB.Error
	}
	return user.ID, nil
}

// 支持如果没有用户则报错
func (d *DBService) UpdateUser(id string, req UserRequest) error {
	if req.Passwd != nil {
		// base64加密
		req.Passwd = tea.String(base64.StdEncoding.EncodeToString([]byte(*req.Passwd)))
	}
	return d.DB.Model(&User{}).Where("id = ?", id).Updates(req).Error
}

func (d *DBService) PatchUserGroup(id string, req *UserPatchMut) error {
	// 先依据 id查到用户
	var user User
	err := d.DB.Model(&User{}).Where("id = ?", id).First(&user).Error
	if err != nil {
		return err
	}
	user.Groups = append(user.Groups, req.Groups...)
	return d.DB.Model(&user).Where("id = ?", id).Update("groups", user.Groups).Error
}
