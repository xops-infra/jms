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

func (d *PolicyService) NeedApprove(username string) ([]*Policy, error) {
	// 是否 admin组，且有需要审批的策略
	var policies []*Policy
	user, err := d.describeUser(username)
	if err != nil {
		return nil, err
	}
	if user.Groups.Contains("admin") {
		if err := d.DB.Where("is_enabled = ?", false).Where("approver is null").Find(&policies).Error; err != nil {
			return nil, err
		}
	}
	return policies, nil
}

func (d *PolicyService) DescribeUser(name string) (*User, error) {
	var user User
	if err := d.DB.Where("username = ?", name).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
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

func (d *PolicyService) UpdateUser(req *UserRequest) error {
	return d.DB.Model(&User{}).Where("email = ?", req.Email).Update("groups", req.Groups).Error
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

func (d *PolicyService) describeUser(name string) (*User, error) {
	var user User
	if err := d.DB.Where("username = ?", name).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *PolicyService) QueryPolicy(username string) ([]*Policy, error) {
	// 查询用户信息
	user, err := d.describeUser(username)
	if err != nil {
		return nil, err
	}
	// 查询用户策略
	return d.queryPolicyByUserAndGroup(username, user.Groups)
}

// 依据用户和组查询策略
func (d *PolicyService) queryPolicyByUserAndGroup(user string, groups utils.ArrayString) ([]*Policy, error) {
	var policies []*Policy
	sql := d.DB.Model(&Policy{})
	sql.Where("is_enabled = ?", true)
	if err := sql.Find(&policies).Error; err != nil {
		return nil, err
	}
	var matchPolicies []*Policy
	for _, policy := range policies {
		// 判断策略是否启用
		if policy.IsEnabled == nil || !*policy.IsEnabled {
			continue
		}
		// 判断策略是否过期
		if policy.IsExpired() {
			// 失效删除
			err := d.DeletePolicy(*policy.Name)
			if err != nil {
				log.Errorf("delete expired policy %s error: %v", *policy.Name, err)
			}
			continue
		}

		var isOk bool
		// 判断用户是否在策略中
		if policy.Users != nil {
			if policy.Users.Contains(user) {
				isOk = true
			}
		}
		if !isOk {
			log.Debugf("user %s not in policy %s", user, *policy.Name)
		}
		// 判断用户所在组是否在策略中
		for _, group := range groups {
			if policy.Groups.Contains(group.(string)) {
				isOk = true
				break
			}
		}
		if !isOk {
			log.Debugf("group %s not in policy %s", groups, *policy.Name)
			continue
		}
		matchPolicies = append(matchPolicies, policy)
	}
	return matchPolicies, nil
}
