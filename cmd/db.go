/*
Copyright © 2024 zhoushoujian <zhoushoujianwork@163.com>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xops-infra/jms/app"
	appConfig "github.com/xops-infra/jms/model"
	"github.com/xops-infra/jms/utils"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	targetEndpoint string // postgresql://username:password@localhost:5432/databasename
	talbeNames     []string
	force          bool
)

// dbCmd represents the db command
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "for db modification",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.Help()
			return
		}
		appConfig.LoadYaml(config)
		err := os.MkdirAll(utils.FilePath(logDir), 0755)
		if err != nil {
			log.Fatalf("create log dir failed: %s", err.Error())
		}
		// init app
		_app := app.NewApp(debug, logDir, rootCmd.Version)
		_app.WithDB(false)
		switch args[0] {
		case "upgrade":
			// 数据结构变更做的库数据升级用,比如这里的 serverfilter数据结构重新设计后字段解析都要重做。
			policies, err := _app.JmsDBService.QueryAllPolicyOld()
			if err != nil {
				log.Fatalf("query all policy failed: %s", err.Error())
			}

			for _, policy := range policies {
				err := _app.JmsDBService.UpdatePolicy(policy.ID, &appConfig.PolicyRequest{
					ServerFilterV1: policy.ServerFilter.ToV1(),
				})
				if err != nil {
					log.Fatalf("update policy %s failed: %s", policy.Name, err.Error())
				}
			}
			log.Infof("all policy upgrade success")
		case "sync":
			if len(talbeNames) == 0 {
				log.Fatalf("need table name, e.g. --table=all, --table=talbe1, --table=talbe2")
			}
			if targetEndpoint == "" {
				log.Fatalf("need target endpoint, e.g. --target-endpoint=postgresql://username:password@localhost:5432/databasename")
			}
			if !_app.Config.WithDB.Enable {
				log.Fatalf("check your config! not enable db")
			}
			// 校验是否是同一个库，是直接报错退出
			if strings.Contains(targetEndpoint, fmt.Sprintf("%s:%d/%s", _app.Config.WithDB.PG.Host, _app.Config.WithDB.PG.Port, _app.Config.WithDB.PG.Database)) {
				log.Fatalf("target endpoint is same with source endpoint, exit")
			}
			if !force {
				// 提醒确认，目标库的表会被删除
				log.Warnf("target endpoint: %s, table: %s, will be deleted&&recreated! continue?\n", targetEndpoint, strings.Join(talbeNames, ","))
				// 提醒用户自行备份输入 Y 继续
				fmt.Printf("input y to continue:\t")
				var input string
				fmt.Scanln(&input)
				if input != "y" && input != "yes" {
					log.Infof("input %s,exit", input)
					return
				}
			}

			// 主数据库同步到备份数据库
			gormConfig := &gorm.Config{}
			if !_app.Debug {
				gormConfig.Logger = logger.Default.LogMode(logger.Silent)
			} else {
				log.Infof("enable gorm log")
				gormConfig.Logger = logger.Default.LogMode(logger.Info)
			}
			rdb, err := gorm.Open(postgres.Open(targetEndpoint), gormConfig)
			if err != nil {
				log.Fatalf("open target db failed: %s", err.Error())
			}

			err = _app.JmsDBService.SyncToTargetDB(rdb, talbeNames)
			if err != nil {
				log.Fatalf("sync db failed: %s", err.Error())
			}
			log.Infof("sync db success")
		}
	},
}

func init() {
	rootCmd.AddCommand(dbCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dbCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// dbCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	dbCmd.Flags().StringVarP(&targetEndpoint, "target-endpoint", "e", "", "target endpoint postgresql://username:password@localhost:5432/databasename")
	dbCmd.Flags().StringSliceVarP(&talbeNames, "table", "t", nil, "table name")
	dbCmd.Flags().BoolVarP(&force, "force", "f", false, "force recreate table")
}
