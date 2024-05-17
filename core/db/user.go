package db

import (
	"encoding/base64"

	"github.com/alibabacloud-go/tea/tea"
	. "github.com/xops-infra/jms/config"
)

// login,
func (d *DBService) Login(username, password string) (bool, error) {
	var user User
	if err := d.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return false, err
	}
	// bas64 加密后比较
	base64Pass := base64.StdEncoding.EncodeToString([]byte(password))
	return base64Pass == tea.StringValue(user.Passwd), nil
}
