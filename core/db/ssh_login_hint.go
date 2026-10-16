package db

import (
	"errors"
	"time"

	"github.com/xops-infra/jms/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *DBService) SSHLoginUsername(asset, address, fingerprint string) (string, error) {
	var hint model.SSHLoginHint
	err := d.DB.Where("asset_id = ? AND key_fingerprint = ? AND address = ?", asset, fingerprint, address).First(&hint).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return hint.Username, err
}

func (d *DBService) RememberSSHLogin(asset, address, fingerprint, username string) error {
	return d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "asset_id"}, {Name: "key_fingerprint"}},
		DoUpdates: clause.AssignmentColumns([]string{"address", "username", "updated_at"}),
	}).Create(&model.SSHLoginHint{AssetID: asset, Address: address, KeyFingerprint: fingerprint, Username: username, UpdatedAt: time.Now()}).Error
}

func (d *DBService) ForgetSSHLogin(asset, address, fingerprint, username string) error {
	return d.DB.Where("asset_id = ? AND key_fingerprint = ? AND address = ? AND username = ?", asset, fingerprint, address, username).Delete(&model.SSHLoginHint{}).Error
}
