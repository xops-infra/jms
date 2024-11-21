/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	"github.com/xops-infra/noop/log"

	"github.com/google/gops/agent"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/api"
	"github.com/xops-infra/jms/core/dingtalk"
	appConfig "github.com/xops-infra/jms/model"
	"github.com/xops-infra/jms/utils"
)

var apiPort int

// apiCmd represents the api command
// @title           cbs manager API
// @version         v1
// @termsOfService  http://swagger.io/terms/
// @host            localhost:8013
// @BasePath
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "api server",
	Long: `api server for jms, must withDB
	swagger url: http://localhost:8013/swagger/index.html
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := agent.Listen(agent.Options{}); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start gops agent: %v\n", err)
			os.Exit(1)
		}
		defer agent.Close()

		appConfig.LoadYaml(config)
		log.Default().WithLevel(log.InfoLevel).WithHumanTime(time.Local).WithFilename(strings.TrimSuffix(logDir, "/") + "/api.log").Init()
		if debug {
			log.Default().WithLevel(log.DebugLevel).WithHumanTime(time.Local).WithFilename(strings.TrimSuffix(logDir, "/") + "/api.log").Init()
			log.Debugf("debug mode, disabled scheduler")
		} else {
			gin.SetMode(gin.ReleaseMode)
		}
		err := os.MkdirAll(utils.FilePath(logDir), 0755)
		if err != nil {
			log.Fatalf("create log dir failed: %s", err.Error())
		}

		// init app
		_app := app.NewApiApplication(debug)

		if app.App.Config.WithSSHCheck.Enable {
			log.Infof("enable dingtalk")
			_app.WithRobot()
		}

		if app.App.Config.WithDB.Enable {
			_app.WithDB(true)
			log.Infof("enable db")
		}

		if app.App.Config.WithDingtalk.Enable {
			log.Infof("enable dingtalk")
			_app.WithDingTalk()
			if !app.App.Config.WithDB.Enable {
				app.App.Config.WithDingtalk.Enable = false
				log.Warnf("dingtalk enable but db not enable, disable dingtalk")
			} else {
				log.Infof("enable api dingtalk Approve")
			}
		}

		app.App.WithMcs()

		go func() {
			for {
				app.App.Core.InstanceIO.LoadServer() // 加载服务列表
				time.Sleep(1 * time.Minute)          // 休眠 1 分钟
			}
		}()

		if !debug {
			// 服务启动后再启动定时任务
			go startApiScheduler()
		}

		log.Infof("api server start on port: %d", apiPort)
		g := api.NewGin()
		log.Errorf(g.Run(fmt.Sprintf(":%d", apiPort)).Error())
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// apiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// apiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	apiCmd.Flags().IntVar(&apiPort, "port", 8013, "api port")
	apiCmd.Flags().StringVar(&logDir, "log-dir", "/opt/jms/logs/", "log dir")
}

// debug will not run
func startApiScheduler() {
	c := cron.New()
	time.Sleep(10 * time.Second) // 等待app初始化完成

	if app.App.Config.WithDB.Enable {
		log.Infof("enabled db config hot update, 2 min check once")
		// 启用定时热加载数据库配置,每 30s 检查一次
		c.AddFunc("*/30 * * * * *", func() {
			app.App.WithMcs()
		})
	}

	if app.App.Config.WithDingtalk.Enable {
		c.AddFunc("0 0 2 * * *", func() {
			err := dingtalk.LoadUsers()
			if err != nil {
				log.Error(err.Error())
			}
		})
		// 定时获取审批列表状态
		c.AddFunc("0 * * * * *", func() {
			dingtalk.LoadApproval()
		})
	}

	c.Start()
	select {}
}
