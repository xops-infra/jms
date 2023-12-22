package policy

import (
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/xops-infra/jms/utils"
)

type PolicyService struct {
	DB *gorm.DB
}

func NewPolicyService(db *gorm.DB) *PolicyService {
	return &PolicyService{
		DB: db,
	}
}

func (d *PolicyService) NeedApprove(username string) ([]*Policy, error) {
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

func (d *PolicyService) DescribeUser(name string) (User, error) {
	var user User
	if err := d.DB.Where("username = ?", name).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func (d *PolicyService) CreateGroup(group *Group) (string, error) {
	// 判断组是否存在
	var count int64
	if err := d.DB.Model(&Group{}).Where("name = ?", group.Name).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("group already exists")
	}

	group.Id = uuid.NewString()
	if d.DB.Create(group).Error != nil {
		return "", d.DB.Error
	}
	return group.Id, nil
}

func (d *PolicyService) DeleteGroup(name string) error {
	return d.DB.Update("is_deleted", true).Where("name = ?", name).Error
}

// 自带校验是否存在
func (d *PolicyService) CreateUser(req *UserRequest) (string, error) {
	user := &User{
		Username: req.Name,
		Email:    req.Email,
		Groups:   req.Groups,
	}
	// 判断用户是否存在
	var count int64
	if err := d.DB.Model(&User{}).Where("username = ?", req.Name).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("user already exists")
	}

	user.Id = uuid.NewString()
	if d.DB.Create(user).Error != nil {
		return "", d.DB.Error
	}
	return user.Id, nil
}

// 支持如果没有用户则报错
func (d *PolicyService) UpdateUserGroups(username string, groups utils.ArrayString) error {
	user, err := d.DescribeUser(username)
	if err != nil {
		return err
	}
	return d.DB.Model(&user).Update("groups", groups).Error
}

func (d *PolicyService) CreatePolicy(req *CreatePolicyRequest) (string, error) {
	// 判断策略是否存在
	var count int64
	if err := d.DB.Model(&Policy{}).Where("name = ?", *req.Name).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("policy already exists")
	}
	newPolicy := &Policy{
		Id:           uuid.NewString(),
		Name:         req.Name,
		IsEnabled:    tea.Bool(false), // 默认不启用，需要审批
		Users:        req.Users,
		Groups:       req.Groups,
		Actions:      req.Actions,
		ServerFilter: req.ServerFilter,
		ExpiresAt:    req.ExpiresAt,
	}
	if d.DB.Create(newPolicy).Error != nil {
		return "", d.DB.Error
	}
	return newPolicy.Id, nil
}

func (d *PolicyService) DeletePolicy(name string) error {
	return d.DB.Where("name = ?", name).UpdateColumn("is_deleted", true).Error
}

func (d *PolicyService) ApprovePolicy(policyName, Approver string, IsEnabled bool) error {
	return d.DB.Model(&Policy{}).Where("name = ?", policyName).Updates(map[string]interface{}{
		"is_enabled": IsEnabled,
		"approver":   Approver,
	}).Error
}

func (d *PolicyService) AddUsersToPolicy(name string, usernames []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("users", gorm.Expr("json_array_append(users, ?)", usernames)).Error
}

func (d *PolicyService) RemoveUsersFromPolicy(name string, usernames []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("users", gorm.Expr("json_remove(users, ?)", usernames)).Error
}

func (d *PolicyService) AddGroupsToPolicy(name string, groups []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("groups", gorm.Expr("json_array_append(groups, ?)", groups)).Error
}

// RemoveGroupsFromPolicy
func (d *PolicyService) RemoveGroupsFromPolicy(name string, groups []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("groups", gorm.Expr("json_remove(groups, ?)", groups)).Error
}

func (d *PolicyService) UpdateActionsOfPolicy(name string, actions []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("actions", actions).Error
}

// 只查询用户的策略
func (d *PolicyService) QueryPolicyByUser(username string) ([]*Policy, error) {
	sql := d.DB.Model(&Policy{}).Where("is_enabled = ?", true)
	tx := sql.Where("users like ?", fmt.Sprintf("%%%s%%", username))
	var policies []*Policy
	if err := tx.Find(&policies).Error; err != nil {
		return nil, err
	}
	var matchPolicies []*Policy
	// 精确返回
	for _, policy := range policies {
		if policy.Users.Contains(username) {
			matchPolicies = append(matchPolicies, policy)
		}
	}
	return matchPolicies, nil
}

// 查询用户组的策略
func (d *PolicyService) QueryPolicyByGroup(group string) ([]*Policy, error) {
	sql := d.DB.Model(&Policy{}).Where("is_enabled = ?", true)
	tx := sql.Where("groups like ?", fmt.Sprintf("%%%s%%", group))
	var policies []*Policy
	if err := tx.Find(&policies).Error; err != nil {
		return nil, err
	}
	var matchPolicies []*Policy
	// 精确返回
	for _, policy := range policies {
		if policy.Groups.Contains(group) {
			matchPolicies = append(matchPolicies, policy)
		}
	}
	return matchPolicies, nil
}

// 查询用户和用户组的策略
func (d *PolicyService) QueryPolicyWithGroup(username string) ([]*Policy, error) {
	var policies []*Policy
	// 查询用户信息
	userPolicies, err := d.QueryPolicyByUser(username)
	if err != nil {
		return nil, err
	}
	policies = append(policies, userPolicies...)

	// 查询用户信息获取组附加组策略
	user, err := d.DescribeUser(username)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		} else {
			return policies, nil
		}
	}
	for _, group := range user.Groups {
		groupPolicies, err := d.QueryPolicyByGroup(group.(string))
		if err != nil {
			return nil, err
		}
		policies = append(policies, groupPolicies...)
	}

	return policies, nil
}
