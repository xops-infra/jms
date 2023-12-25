package policy

import (
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/noop/log"
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

// InitDefault
func (d *PolicyService) InitDefault() error {
	// 创建 admin 组
	group := &Group{
		Id:   uuid.NewString(),
		Name: tea.String("admin"),
	}
	if err := d.DB.Create(group).Error; err != nil {
		return err
	}

	// 创建 admin 用户
	user := &User{
		Id:       uuid.NewString(),
		Username: tea.String("admin"),
		Email:    tea.String("admin@example.com"),
		Groups:   utils.ArrayString{"admin"},
		Passwd:   utils.GeneratePasswd("admin"),
	}
	if err := d.DB.Create(user).Error; err != nil {
		return err
	}
	return nil
}

// login,
func (d *PolicyService) Login(username, password string) (bool, error) {
	var user User
	if err := d.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return false, err
	}
	if utils.CheckPasswd(password, string(user.Passwd)) {
		return true, nil
	}
	return false, nil
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
func (d *PolicyService) CreateUser(req *CreateUserRequest) (string, error) {
	user := &User{
		Username: req.Name,
		Email:    req.Email,
		Groups:   req.Groups,
		Passwd:   utils.GeneratePasswd(*req.Passwd),
	}
	if req.IsLdap != nil {
		user.IsLdap = req.IsLdap
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

func (d *PolicyService) CreatePolicy(req *PolicyMut) (string, error) {
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

func (d *PolicyService) UpdatePolicy(id string, mut *PolicyMut) error {
	policy, err := d.QueryPolicyById(id)
	if err != nil {
		return err
	}
	return d.DB.Model(policy).Updates(mut).Error
}

func (d *PolicyService) DeletePolicy(id string) error {
	return d.DB.Where("id = ?", id).UpdateColumn("is_deleted", true).Error
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
func (d *PolicyService) QueryPolicyByUser(username string) ([]Policy, error) {
	sql := d.DB.Model(&Policy{}).Where("is_deleted = ?", false)
	tx := sql.Where("users like ?", fmt.Sprintf("%%%s%%", username))
	var policies []Policy
	if err := tx.Find(&policies).Error; err != nil {
		return nil, err
	}
	var matchPolicies []Policy
	// 精确返回
	for _, policy := range policies {
		if policy.Users.Contains(username) {
			log.Debugf("policy: %v", policy)
			matchPolicies = append(matchPolicies, policy)
		}
	}
	return matchPolicies, nil
}

// 查询用户组的策略
func (d *PolicyService) QueryPolicyByGroup(group string) ([]Policy, error) {
	sql := d.DB.Model(&Policy{}).Where("is_deleted = ?", false)
	tx := sql.Where("groups like ?", fmt.Sprintf("%%%s%%", group))
	var policies []Policy
	if err := tx.Find(&policies).Error; err != nil {
		return nil, err
	}
	var matchPolicies []Policy
	// 精确返回
	for _, policy := range policies {
		if policy.Groups.Contains(group) {
			matchPolicies = append(matchPolicies, policy)
		}
	}
	return matchPolicies, nil
}

// 查询用户和用户组的策略
func (d *PolicyService) QueryPolicyWithGroup(username string) ([]Policy, error) {
	var policies []Policy
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

// 查询策略名称
func (d *PolicyService) QueryPolicyByName(name string) ([]Policy, error) {
	sql := d.DB.Model(&Policy{}).Where("is_deleted = ?", false)
	if name != "" {
		sql = sql.Where("name = ?", name)
	}
	var policies []Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

// QueryPolicyById
func (d *PolicyService) QueryPolicyById(id string) (*Policy, error) {
	var policy Policy
	if err := d.DB.Where("id = ?", id).First(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

// 查询所有
func (d *PolicyService) QueryAllPolicy() ([]Policy, error) {
	sql := d.DB.Model(&Policy{}).Where("is_deleted = ?", false)
	var policies []Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}
