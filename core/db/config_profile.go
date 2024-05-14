package db

import (
	"encoding/base64"
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/noop/log"
)

func (d *DBService) ListProfile() ([]config.Profile, error) {
	var profiles []config.Profile
	err := d.DB.Where("is_delete is false").Find(&profiles).Order("created_at").Error
	// 隐藏敏感信息
	for i := range profiles {
		profiles[i].SK = "****"
	}
	return profiles, err
}

// 内部服务调用，不隐藏敏感信息
func (d *DBService) LoadProfile() ([]config.CreateProfileRequest, error) {
	var profiles []config.Profile
	err := d.DB.Where("is_delete is false").Find(&profiles).Error
	if err != nil {
		return nil, err
	}
	var reqs []config.CreateProfileRequest
	// base64 解密
	for i := range profiles {
		sk, err := base64.StdEncoding.DecodeString(profiles[i].SK)
		if err != nil {
			return nil, fmt.Errorf("base64 decode error: %v", err)
		}
		reqs = append(reqs, config.CreateProfileRequest{
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

func (d *DBService) CreateProfile(req config.CreateProfileRequest) (string, error) {
	if req.Name == nil || req.AK == nil || req.SK == nil || req.Cloud == nil || len(req.Regions) == 0 {
		return "", fmt.Errorf("name, ak, sk, cloud, regions are required")
	}
	// 先查询是否存在
	var count int64
	err := d.DB.Model(config.Profile{}).Where("name = ?", *req.Name).Where("is_delete is false").Count(&count).Error
	if err != nil {
		return "", fmt.Errorf("query profile error: %v", err)
	}
	if count > 0 {
		return "", fmt.Errorf("profile name %s already exists", *req.Name)
	}
	// SK base64 加密
	baseSK := base64.StdEncoding.EncodeToString([]byte(*req.SK))
	profile := config.Profile{
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

func (d *DBService) UpdateProfile(uuid string, req config.CreateProfileRequest) error {
	profile := config.Profile{}
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
	err := d.DB.Model(config.Profile{}).Where("uuid = ?", uuid).Updates(&profile).Error
	return err
}

func (d *DBService) DeleteProfile(uuid string) error {
	// 先查询是否存在
	var profile config.Profile
	err := d.DB.Where("uuid = ?", uuid).Where("is_delete is false").First(&profile).Error
	if err != nil {
		return err
	}
	return d.DB.Model(config.Profile{}).Where("uuid = ?", uuid).Update("is_delete", true).Error
}
