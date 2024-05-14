package db

import (
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/jms/config"
	"gorm.io/gorm"
)

func (d *DBService) CreatePolicy(req *config.PolicyMut, approval_id *string) (string, error) {
	// 判断策略是否存在
	var count int64
	if err := d.DB.Model(&config.Policy{}).Where("name = ?", *req.Name).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("policy already exists")
	}
	newPolicy := &config.Policy{
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

func (d *DBService) UpdatePolicy(id string, mut *config.PolicyMut) error {
	policy, err := d.QueryPolicyById(id)
	if err != nil {
		return err
	}
	return d.DB.Model(policy).Updates(mut).Error
}

func (d *DBService) UpdatePolicyStatus(id string, mut config.ApprovalResult) error {
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
	return d.DB.Model(&config.Policy{}).Where("name = ?", policyName).Updates(map[string]interface{}{
		"is_enabled": IsEnabled,
		"approver":   Approver,
	}).Error
}

func (d *DBService) AddUsersToPolicy(name string, usernames []string) error {
	return d.DB.Model(&config.Policy{}).Where("name = ?", name).Update("users", gorm.Expr("json_array_append(users, ?)", usernames)).Error
}

func (d *DBService) RemoveUsersFromPolicy(name string, usernames []string) error {
	return d.DB.Model(&config.Policy{}).Where("name = ?", name).Update("users", gorm.Expr("json_remove(users, ?)", usernames)).Error
}

func (d *DBService) AddGroupsToPolicy(name string, groups []string) error {
	return d.DB.Model(&config.Policy{}).Where("name = ?", name).Update("groups", gorm.Expr("json_array_append(groups, ?)", groups)).Error
}

// RemoveGroupsFromPolicy
func (d *DBService) RemoveGroupsFromPolicy(name string, groups []string) error {
	return d.DB.Model(&config.Policy{}).Where("name = ?", name).Update("groups", gorm.Expr("json_remove(groups, ?)", groups)).Error
}

func (d *DBService) UpdateActionsOfPolicy(name string, actions []string) error {
	return d.DB.Model(&config.Policy{}).Where("name = ?", name).Update("actions", actions).Error
}

// 只查询用户的策略
// 支持policy users 包含*的情况，表示都能命中
func (d *DBService) QueryPolicyByUser(username string) ([]config.Policy, error) {
	sql := d.DB.Model(&config.Policy{}).Where("is_deleted = ?", false)
	var policies []config.Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	var matchPolicies []config.Policy
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
// 	sql := d.DB.Model(&config.Policy{}).Where("is_deleted = ?", false)
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
func (d *DBService) QueryPolicyByName(name string) ([]config.Policy, error) {
	sql := d.DB.Model(&config.Policy{}).Where("is_deleted = ?", false)
	if name != "" {
		sql = sql.Where("name = ?", name)
	}
	var policies []config.Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

// QueryPolicyById
func (d *DBService) QueryPolicyById(id string) (*config.Policy, error) {
	var policy config.Policy
	if err := d.DB.Where("id = ?", id).First(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

// 查询所有
func (d *DBService) QueryAllPolicy() ([]config.Policy, error) {
	sql := d.DB.Model(&config.Policy{}).Where("is_deleted = ?", false)
	var policies []config.Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}
