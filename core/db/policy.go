package db

import (
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/jms/model"
	"gorm.io/gorm"
)

func (d *DBService) CreatePolicy(req *model.PolicyRequest, approval_id *string) (string, error) {
	if req.Name == nil || req.ServerFilterV1 == nil || req.ExpiresAt == nil {
		return "", fmt.Errorf("invalid request. please check required fields")
	}
	// 判断策略是否存在
	var count int64
	if err := d.DB.Model(&model.Policy{}).Where("name = ?", *req.Name).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("policy already exists")
	}
	newPolicy := &model.Policy{
		ID:             uuid.NewString(),
		Name:           tea.StringValue(req.Name),
		IsEnabled:      false, // 默认不启用，需要审批
		Users:          req.Users,
		Actions:        req.Actions,
		ServerFilter:   nil,
		ServerFilterV1: req.ServerFilterV1,
		ExpiresAt:      *req.ExpiresAt,
		ApprovalID:     tea.StringValue(approval_id),
	}
	if d.DB.Create(newPolicy).Error != nil {
		return "", d.DB.Error
	}
	return newPolicy.ID, nil
}

func (d *DBService) UpdatePolicy(id string, mut *model.PolicyRequest) error {
	policy, err := d.QueryPolicyById(id)
	if err != nil {
		return err
	}
	return d.DB.Model(policy).Updates(mut).Error
}

func (d *DBService) UpdatePolicyStatus(id string, mut model.ApprovalResult) error {
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
	return d.DB.Model(&model.Policy{}).Where("name = ?", policyName).Updates(map[string]interface{}{
		"is_enabled": IsEnabled,
		"approver":   Approver,
	}).Error
}

func (d *DBService) AddUsersToPolicy(name string, usernames []string) error {
	return d.DB.Model(&model.Policy{}).Where("name = ?", name).Update("users", gorm.Expr("json_array_append(users, ?)", usernames)).Error
}

func (d *DBService) RemoveUsersFromPolicy(name string, usernames []string) error {
	return d.DB.Model(&model.Policy{}).Where("name = ?", name).Update("users", gorm.Expr("json_remove(users, ?)", usernames)).Error
}

func (d *DBService) AddGroupsToPolicy(name string, groups []string) error {
	return d.DB.Model(&model.Policy{}).Where("name = ?", name).Update("groups", gorm.Expr("json_array_append(groups, ?)", groups)).Error
}

// RemoveGroupsFromPolicy
func (d *DBService) RemoveGroupsFromPolicy(name string, groups []string) error {
	return d.DB.Model(&model.Policy{}).Where("name = ?", name).Update("groups", gorm.Expr("json_remove(groups, ?)", groups)).Error
}

func (d *DBService) UpdateActionsOfPolicy(name string, actions []string) error {
	return d.DB.Model(&model.Policy{}).Where("name = ?", name).Update("actions", actions).Error
}

// 只查询用户的策略
// 支持policy users 包含*的情况，表示都能命中
func (d *DBService) QueryPolicyByUser(username string) ([]model.Policy, error) {
	sql := d.DB.Model(&model.Policy{}).Where("is_deleted = ?", false)
	var policies []model.Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	var matchPolicies []model.Policy
	// 精确返回
	for _, policy := range policies {
		if policy.Users.Contains(username) {
			// log.Debugf("policy: %s", tea.Prettify(policy))
			matchPolicies = append(matchPolicies, policy)
		}
	}
	return matchPolicies, nil
}

// 查询策略名称
func (d *DBService) QueryPolicyByName(name string) ([]model.Policy, error) {
	sql := d.DB.Model(&model.Policy{}).Where("is_deleted = ?", false)
	if name != "" {
		sql = sql.Where("name = ?", name)
	}
	var policies []model.Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

// QueryPolicyById
func (d *DBService) QueryPolicyById(id string) (*model.Policy, error) {
	var policy model.Policy
	if err := d.DB.Where("id = ?", id).First(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

// 查询所有
func (d *DBService) QueryAllPolicyOld() ([]model.PolicyOld, error) {
	sql := d.DB.Model(&model.PolicyOld{}).Where("is_deleted = ?", false)
	var policies []model.PolicyOld
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

func (d *DBService) QueryAllPolicy() ([]model.Policy, error) {
	sql := d.DB.Model(&model.Policy{}).Where("is_deleted = ?", false)
	var policies []model.Policy
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}
