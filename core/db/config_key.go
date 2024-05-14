package db

import (
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/jms/config"
)

func (d *DBService) InternalLoadKey() ([]config.AddKeyRequest, error) {
	var keys []config.Key
	err := d.DB.Where("is_delete is false").Find(&keys).Order("created_at").Error
	if err != nil {
		return nil, err
	}
	var res []config.AddKeyRequest
	for i := range keys {
		res = append(res, config.AddKeyRequest{
			IdentityFile: tea.String(keys[i].KeyName),
			PemBase64:    tea.String(keys[i].PemBase64),
			KeyID:        tea.String(keys[i].KeyID),
			Profile:      tea.String(keys[i].Profile),
		})
	}
	return res, nil
}

func (d *DBService) ListKey() ([]config.Key, error) {
	var keys []config.Key
	err := d.DB.Where("is_delete is false").Find(&keys).Order("created_at").Error
	// 隐藏敏感信息
	for i := range keys {
		keys[i].PemBase64 = "****"
	}
	return keys, err
}

// 支持判断 key_id 是否存在
func (d *DBService) AddKey(req config.AddKeyRequest) (string, error) {
	if req.IdentityFile == nil || req.PemBase64 == nil || req.KeyID == nil || req.Profile == nil {
		return "", fmt.Errorf("invalid request")
	}
	if !strings.HasSuffix(*req.IdentityFile, ".pem") {
		return "", fmt.Errorf("invalid identity_file(private key file name), must end with .pem, casue you download from cloud auto has .pem")
	}
	// 先查询是否存在
	var count int64
	err := d.DB.Model(config.Key{}).Where("key_id = ?", *req.KeyID).Where("is_delete is false").Count(&count).Error
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("key_id %s already exists", tea.StringValue(req.KeyID))
	}
	key := &config.Key{
		IsDelete:  false,
		UUID:      uuid.NewString(),
		KeyID:     tea.StringValue(req.KeyID),
		KeyName:   tea.StringValue(req.IdentityFile),
		Profile:   tea.StringValue(req.Profile),
		PemBase64: tea.StringValue(req.PemBase64),
	}
	return key.UUID, d.DB.Create(key).Error
}

func (d *DBService) DeleteKey(uuid string) error {
	// 先查询是否存在
	var key config.Key
	err := d.DB.Where("uuid = ?", uuid).Where("is_delete is false").First(&key).Error
	if err != nil {
		return err
	}
	return d.DB.Model(&key).Update("is_delete", true).Error
}
