package db

import (
	"encoding/base64"
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/noop/log"
	"gorm.io/gorm"
)

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

func (d *DBService) ListProfile() ([]Profile, error) {
	var profiles []Profile
	err := d.DB.Where("is_delete is false").Find(&profiles).Order("created_at").Error
	// 隐藏敏感信息
	for i := range profiles {
		profiles[i].SK = "****"
	}
	return profiles, err
}

// 内部服务调用，不隐藏敏感信息
func (d *DBService) LoadProfile() ([]CreateProfileRequest, error) {
	var profiles []Profile
	err := d.DB.Where("is_delete is false").Find(&profiles).Error
	if err != nil {
		return nil, err
	}
	var reqs []CreateProfileRequest
	// base64 解密
	for i := range profiles {
		sk, err := base64.StdEncoding.DecodeString(profiles[i].SK)
		if err != nil {
			return nil, fmt.Errorf("base64 decode error: %v", err)
		}
		reqs = append(reqs, CreateProfileRequest{
			Name:    tea.String(profiles[i].Name),
			AK:      tea.String(profiles[i].AK),
			SK:      tea.String(string(sk)),
			Cloud:   tea.String(profiles[i].Cloud),
			Regions: profiles[i].Regions,
			Enabled: profiles[i].Enabled,
		})
	}
	return reqs, nil
}

func (d *DBService) CreateProfile(req CreateProfileRequest) (string, error) {
	if req.Name == nil || req.AK == nil || req.SK == nil || req.Cloud == nil || len(req.Regions) == 0 {
		return "", fmt.Errorf("name, ak, sk, cloud, regions are required")
	}
	// 先查询是否存在
	var count int64
	err := d.DB.Model(Profile{}).Where("name = ?", *req.Name).Where("is_delete is false").Count(&count).Error
	if err != nil {
		return "", fmt.Errorf("query profile error: %v", err)
	}
	if count > 0 {
		return "", fmt.Errorf("profile name %s already exists", *req.Name)
	}
	// SK base64 加密
	baseSK := base64.StdEncoding.EncodeToString([]byte(*req.SK))
	profile := Profile{
		UUID:    uuid.New().String(),
		Name:    *req.Name,
		AK:      *req.AK,
		SK:      baseSK,
		Cloud:   *req.Cloud,
		Regions: req.Regions,
	}
	log.Debugf(tea.Prettify(profile))
	err = d.DB.Create(&profile).Error
	return profile.UUID, err
}

func (d *DBService) UpdateProfile(uuid string, req CreateProfileRequest) error {
	profile := Profile{}
	if req.Name != nil {
		profile.Name = *req.Name
	}
	if req.AK != nil {
		profile.AK = *req.AK
	}
	if req.SK != nil {
		baseSK := base64.StdEncoding.EncodeToString([]byte(*req.SK))
		profile.SK = baseSK
	}
	if req.Cloud != nil {
		profile.Cloud = *req.Cloud
	}
	if len(req.Regions) > 0 {
		profile.Regions = req.Regions
	}
	err := d.DB.Model(Profile{}).Where("uuid = ?", uuid).Updates(&profile).Error
	return err
}

func (d *DBService) DeleteProfile(uuid string) error {
	// 先查询是否存在
	var profile Profile
	err := d.DB.Where("uuid = ?", uuid).Where("is_delete is false").First(&profile).Error
	if err != nil {
		return err
	}
	return d.DB.Model(Profile{}).Where("uuid = ?", uuid).Update("is_delete", true).Error
}
