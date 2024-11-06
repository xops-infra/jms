package db

import (
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/noop/log"
	"gorm.io/gorm"
)

// SyncToTargetDB 同步到备份数据库
// 会删除备份库的表再创建
func (d *DBService) SyncToTargetDB(targetRdb *gorm.DB, talbeNames []string) error {
	todos := make([]string, 0)
	for _, tableName := range talbeNames {
		if tableName == "all" {
			todos = nil
			break
		} else {
			todos = append(todos, tableName)
		}
	}

	// 全量主数据库同步到备份数据库
	// 1. 获取主库所有表
	tables, err := d.DB.Migrator().GetTables()
	if err != nil {
		return fmt.Errorf("source db get tables failed: %v", err)
	}

	if len(todos) == 0 {
		todos = tables
	}

	// 2. 开始同步 todos表
	for _, table := range todos {
		// 获取源表数据
		rows, err := d.DB.Table(table).Rows()
		if err != nil {
			return fmt.Errorf("source db get rows failed: %v", err)
		}
		defer rows.Close()

		// 清空目标表数据
		if err := targetRdb.Exec(fmt.Sprintf("delete from %s", table)).Error; err != nil {
			return fmt.Errorf("target db delete table failed: %v", err)
		}

		createCount := 0
		for rows.Next() {
			var row map[string]interface{}
			if err := d.DB.Table(table).ScanRows(rows, &row); err != nil {
				return fmt.Errorf("source db scan rows failed: %v", err)
			}
			if err := targetRdb.Table(table).Create(row).Error; err != nil {
				return fmt.Errorf("target db create failed: %v", err)
			}
			createCount++
			log.Debugf("create table %s data %s", table, tea.Prettify(row))
		}
		log.Infof("sync table %s, create count %d", table, createCount)
	}

	return nil
}
