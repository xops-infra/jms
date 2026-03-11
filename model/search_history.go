package model

import "gorm.io/gorm"

// SearchHistory stores recent menu search keywords for a user.
type SearchHistory struct {
	gorm.Model
	User    string `json:"user" gorm:"column:user;type:varchar(255);not null"`
	Keyword string `json:"keyword" gorm:"column:keyword;type:varchar(255);not null"`
}

func (SearchHistory) TableName() string {
	return "pui_search_history"
}
