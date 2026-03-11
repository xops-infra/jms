package db

import (
	"strings"

	"github.com/xops-infra/jms/model"
)

const defaultSearchHistoryLimit = 20

// AddSearchHistory saves a keyword for the given user and keeps the latest N records.
func (d *DBService) AddSearchHistory(user, keyword string, limit int) error {
	keyword = strings.TrimSpace(keyword)
	if user == "" || keyword == "" {
		return nil
	}
	if limit <= 0 {
		limit = defaultSearchHistoryLimit
	}
	// remove duplicate keyword for this user before inserting
	if err := d.DB.Unscoped().
		Where("user = ? AND keyword = ?", user, keyword).
		Delete(&model.SearchHistory{}).Error; err != nil {
		return err
	}
	if err := d.DB.Create(&model.SearchHistory{
		User:    user,
		Keyword: keyword,
	}).Error; err != nil {
		return err
	}

	var extras []model.SearchHistory
	if err := d.DB.Where("user = ?", user).
		Order("created_at desc").
		Offset(limit).
		Find(&extras).Error; err != nil {
		return err
	}
	if len(extras) == 0 {
		return nil
	}
	return d.DB.Unscoped().Delete(&extras).Error
}

// ListSearchHistory returns latest N records ordered by created_at desc.
func (d *DBService) ListSearchHistory(user string, limit int) ([]model.SearchHistory, error) {
	user = strings.TrimSpace(user)
	if user == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchHistoryLimit
	}
	var records []model.SearchHistory
	err := d.DB.Where("user = ?", user).
		Order("created_at desc").
		Limit(limit).
		Find(&records).Error
	return records, err
}
