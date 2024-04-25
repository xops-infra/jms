package db

import (
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddKeyRequest struct {
	IdentityFile *string `json:"identity_file" mapstructure:"identity_file"`              // 云上下载下来的名字，比如 jms-key.pem
	PemBase64    *string `json:"pem_base64" binding:"required" mapstructure:"pem_base64"` // base64
	KeyID        *string `json:"key_id" binding:"required" mapstructure:"key_id"`         // 云上的key id，比如 skey-123456
	Profile      *string `json:"profile"`                                                 // 云账号的 profile，比如 aws, aliyun
}

type Key struct {
	gorm.Model `json:"-"`
	IsDelete   bool   `gorm:"column:is_delete;type:boolean;not null;default:false"`
	UUID       string `gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	KeyID      string `gorm:"column:key_id;type:varchar(36);unique_index;not null"`
	KeyName    string `gorm:"column:key_name;type:varchar(255);unique_index;not null"`
	Profile    string `gorm:"column:profile;type:varchar(255);not null"`
	PemBase64  string `gorm:"column:pem_base64;type:text;not null"`
}

func (Key) TableName() string {
	return "key_table"
}

func (d *DBService) InternalLoad() ([]AddKeyRequest, error) {
	var keys []Key
	err := d.DB.Where("is_delete is false").Find(&keys).Order("created_at").Error
	if err != nil {
		return nil, err
	}
	var res []AddKeyRequest
	for i := range keys {
		res = append(res, AddKeyRequest{
			IdentityFile: tea.String(keys[i].KeyName),
			PemBase64:    tea.String(keys[i].PemBase64),
			KeyID:        tea.String(keys[i].KeyID),
			Profile:      tea.String(keys[i].Profile),
		})
	}
	return res, nil
}

func (d *DBService) ListKey() ([]Key, error) {
	var keys []Key
	err := d.DB.Where("is_delete is false").Find(&keys).Order("created_at").Error
	// 隐藏敏感信息
	for i := range keys {
		keys[i].PemBase64 = "****"
	}
	return keys, err
}

// 支持判断 keyname 是否存在
func (d *DBService) AddKey(req AddKeyRequest) (string, error) {
	// 先查询是否存在
	var count int64
	err := d.DB.Model(Key{}).Where("key_name = ?", tea.StringValue(req.IdentityFile)).Where("is_delete is false").Count(&count).Error
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", fmt.Errorf("key name %s already exists", tea.StringValue(req.IdentityFile))
	}
	key := &Key{
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
	var key Key
	err := d.DB.Where("uuid = ?", uuid).Where("is_delete is false").First(&key).Error
	if err != nil {
		return err
	}
	return d.DB.Model(&key).Update("is_delete", true).Error
}
