package utils

import (
	"database/sql/driver"
	"encoding/json"

	"golang.org/x/crypto/bcrypt"
)

// 用来存储json数组，gorm默认不支持

type ArrayString []interface{}

func (a ArrayString) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ArrayString) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}

func (a ArrayString) Contains(value string) bool {
	for _, item := range a {
		if item == value {
			return true
		}
	}
	return false
}

// 可以预定义一些资产用来快速分配给其他策略c
type ServerFilter struct {
	Name    *string `json:"name" `
	IpAddr  *string `json:"ip_addr" `
	EnvType *string `json:"env_type"`
	Team    *string `json:"team"`
}

func (a ServerFilter) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ServerFilter) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}

func GeneratePasswd(passwd string) []byte {
	// 使用bcrypt生成哈希密码
	hash, _ := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	return hash
}

func CheckPasswd(hashPasswd, passwd string) bool {
	// 检查密码是否正确
	err := bcrypt.CompareHashAndPassword([]byte(hashPasswd), []byte(passwd))
	return err == nil
}
