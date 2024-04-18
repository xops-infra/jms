package db

import (
	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddKeyRequest struct {
	KeyName *string `json:"key_name"`                   // 云上下载下来的名字，比如 jms-key.pem
	PemMd5  *string `json:"pem_md5" binding:"required"` // md5
	KeyID   *string `json:"key_id" binding:"required"`  // 云上的key id，比如 skey-123456
	Profile *string `json:"profile"`                    // 云账号的 profile，比如 aws, aliyun
}

type Key struct {
	gorm.Model
	IsDelete bool   `gorm:"column:is_delete;type:boolean;not null;default:false"`
	UUID     string `gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	KeyID    string `gorm:"column:key_id;type:varchar(36);unique_index;not null"`
	KeyName  string `gorm:"column:key_name;type:varchar(255);not null"`
	Profile  string `gorm:"column:profile;type:varchar(255);not null"`
	PemMd5   string `gorm:"column:pem_md5;type:text;not null"`
}

func (Key) TableName() string {
	return "key_table"
}

func (d *DBService) InternalLoad() (map[string]Key, error) {
	var keys []Key
	err := d.DB.Where("is_delete is false").Find(&keys).Order("created_at").Error
	if err != nil {
		return nil, err
	}
	keyMap := make(map[string]Key)
	for _, key := range keys {
		keyMap[key.KeyID] = key
	}
	return keyMap, nil
}

func (d *DBService) ListKey() ([]Key, error) {
	var keys []Key
	err := d.DB.Where("is_delete is false").Find(&keys).Order("created_at").Error
	// 隐藏敏感信息
	for i := range keys {
		keys[i].PemMd5 = "****"
	}
	return keys, err
}

func (d *DBService) AddKey(req AddKeyRequest) (string, error) {
	key := &Key{
		IsDelete: false,
		UUID:     uuid.NewString(),
		KeyID:    tea.StringValue(req.KeyID),
		KeyName:  tea.StringValue(req.KeyName),
		Profile:  tea.StringValue(req.Profile),
		PemMd5:   tea.StringValue(req.PemMd5),
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
