package db

import (
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/noop/log"
	"gorm.io/gorm"
)

type DBService struct {
	DB *gorm.DB
}

func NewDbService(db *gorm.DB) *DBService {
	return &DBService{
		DB: db,
	}
}

// InitDefault
func (d *DBService) InitDefault() error {
	// 创建 admin 用户
	user := &User{
		ID:       uuid.NewString(),
		Username: tea.String("admin"),
		Email:    tea.String("admin@example.com"),
		Groups:   ArrayString{tea.String("admin")},
		Passwd:   GeneratePasswd("admin"),
	}
	if err := d.DB.Create(user).Error; err != nil {
		return err
	}
	return nil
}

// login,
func (d *DBService) Login(username, password string) bool {
	var user User
	if err := d.DB.Where("username = ?", username).First(&user).Error; err != nil {
		log.Errorf("login error: %s", err)
		return false
	}
	return CheckPasswd(password, string(user.Passwd))
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
		user.Passwd = GeneratePasswd(*req.Passwd)
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

func (d *DBService) CreatePolicy(req *PolicyMut, approval_id *string) (string, error) {
	// 判断策略是否存在
	var count int64
	if err := d.DB.Model(&Policy{}).Where("name = ?", *req.Name).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("policy already exists")
	}
	newPolicy := &Policy{
		ID:        uuid.NewString(),
		Name:      req.Name,
		IsEnabled: tea.Bool(false), // 默认不启用，需要审批
		Users:     req.Users,
		// Groups:       req.Groups,
		Actions:      req.Actions,
		ServerFilter: req.ServerFilter,
		ExpiresAt:    req.ExpiresAt,
		ApprovalID:   approval_id,
	}
	if d.DB.Create(newPolicy).Error != nil {
		return "", d.DB.Error
	}
	return newPolicy.ID, nil
}

func (d *DBService) UpdatePolicy(id string, mut *PolicyMut) error {
	policy, err := d.QueryPolicyById(id)
	if err != nil {
		return err
	}
	return d.DB.Model(policy).Updates(mut).Error
}

func (d *DBService) UpdatePolicyStatus(id string, mut ApprovalResult) error {
	policy, err := d.QueryPolicyById(id)
	if err != nil {
		return err
	}
	return d.DB.Model(policy).Updates(map[string]interface{}{
		"is_enabled": mut.IsPass,
		"approver":   mut.Applicant,
	}).Error
}

func (d *DBService) DeletePolicy(id string) error {
	return d.DB.Where("id = ?", id).UpdateColumn("is_deleted", true).Error
}

func (d *DBService) ApprovePolicy(policyName, Approver string, IsEnabled bool) error {
	return d.DB.Model(&Policy{}).Where("name = ?", policyName).Updates(map[string]interface{}{
		"is_enabled": IsEnabled,
		"approver":   Approver,
	}).Error
}

func (d *DBService) AddUsersToPolicy(name string, usernames []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("users", gorm.Expr("json_array_append(users, ?)", usernames)).Error
}

func (d *DBService) RemoveUsersFromPolicy(name string, usernames []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("users", gorm.Expr("json_remove(users, ?)", usernames)).Error
}

func (d *DBService) AddGroupsToPolicy(name string, groups []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("groups", gorm.Expr("json_array_append(groups, ?)", groups)).Error
}

// RemoveGroupsFromPolicy
func (d *DBService) RemoveGroupsFromPolicy(name string, groups []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("groups", gorm.Expr("json_remove(groups, ?)", groups)).Error
}

func (d *DBService) UpdateActionsOfPolicy(name string, actions []string) error {
	return d.DB.Model(&Policy{}).Where("name = ?", name).Update("actions", actions).Error
}

// 只查询用户的策略
// 支持policy users 包含*的情况，表示都能命中
func (d *DBService) QueryPolicyByUser(username string) ([]Policy, error) {
	sql := d.DB.Model(&Policy{}).Where("is_deleted = ?", false)
	var policies []Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	var matchPolicies []Policy
	// 精确返回
	for _, policy := range policies {
		if policy.Users.Contains(username) {
			// log.Debugf("policy: %s", tea.Prettify(policy))
			matchPolicies = append(matchPolicies, policy)
		}
	}
	return matchPolicies, nil
}

// // 查询用户组的策略
// func (d *PolicyService) QueryPolicyByGroup(group string) ([]Policy, error) {
// 	sql := d.DB.Model(&Policy{}).Where("is_deleted = ?", false)
// 	tx := sql.Where("groups like ?", fmt.Sprintf("%%%s%%", group))
// 	var policies []Policy
// 	if err := tx.Find(&policies).Error; err != nil {
// 		return nil, err
// 	}
// 	var matchPolicies []Policy
// 	// 精确返回
// 	for _, policy := range policies {
// 		if policy.Groups.Contains(group) {
// 			matchPolicies = append(matchPolicies, policy)
// 		}
// 	}
// 	return matchPolicies, nil
// }

// 查询用户和用户组的策略
// func (d *PolicyService) QueryPolicyWithGroup(username string) ([]Policy, error) {
// 	var policies []Policy
// 	// 查询用户信息
// 	userPolicies, err := d.QueryPolicyByUser(username)
// 	if err != nil {
// 		return nil, err
// 	}
// 	policies = append(policies, userPolicies...)

// 	// 查询用户信息获取组附加组策略
// 	user, err := d.DescribeUser(username)
// 	if err != nil {
// 		if err != gorm.ErrRecordNotFound {
// 			return nil, err
// 		} else {
// 			return policies, nil
// 		}
// 	}
// 	for _, group := range user.Groups {
// 		groupPolicies, err := d.QueryPolicyByGroup(group.(string))
// 		if err != nil {
// 			return nil, err
// 		}
// 		policies = append(policies, groupPolicies...)
// 	}
// 	return policies, nil
// }

// 查询策略名称
func (d *DBService) QueryPolicyByName(name string) ([]Policy, error) {
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
func (d *DBService) QueryPolicyById(id string) (*Policy, error) {
	var policy Policy
	if err := d.DB.Where("id = ?", id).First(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

// 查询所有
func (d *DBService) QueryAllPolicy() ([]Policy, error) {
	sql := d.DB.Model(&Policy{}).Where("is_deleted = ?", false)
	var policies []Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}
