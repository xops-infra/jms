/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core"
	"github.com/xops-infra/jms/core/dingtalk"
	"github.com/xops-infra/jms/io"
	"github.com/xops-infra/noop/log"
)

// SchedulerCmd represents the Scheduler command
var SchedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "run Scheduler, should be only 1 instance",
	Long: `运行一些辅助任务，主要如下功能：
- 执行定时任务，加载一些必要信息，比如如果接入钉钉审批则需要录入人员信息；
- 执行定时任务，检查机器 ssh 可连接性；
- 执行定时任务，加载云服务器信息入库；
- 执行定时任务，检查机器 ssh 可连接性并依据配置发送钉钉告警通知；
- 执行批量脚本；
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Scheduler called")
		_app := app.NewApplication(debug, logDir, rootCmd.Version, config)

		if app.App.Config.WithDB.Enable {
			log.Infof("enable db with automigrate")
			_app.WithDB(true)
			app.App.Sshd.SshdIO = io.NewSshd(app.App.DBIo, app.App.Config.LocalServers.ToMapWithHost()) // todo: 把认证那块的函数移到 db操作
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

		if app.App.Config.WithSSHCheck.Enable {
			log.Infof("enable dingtalk")
			_app.Scheduler.RobotClient = dt.NewRobotClient()
		}

		app.App.WithMcs()

		go func() {
			for {
				app.App.Scheduler.InstanceIO.LoadServer() // 加载服务列表
				time.Sleep(1 * time.Minute)               // 休眠 1 分钟
			}
		}()

		startScheduler()
	},
}

func init() {
	rootCmd.AddCommand(SchedulerCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// SchedulerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// SchedulerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func startScheduler() {
	c := cron.New()
	time.Sleep(10 * time.Second) // 等待app初始化完成

	// 刷云服务器信息
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

	if app.App.Config.WithDB.Enable {
		c.AddFunc("0 * * * * *", func() {
			log.Infof("run shell task")
			err := core.ServerShellRun() // 每 1min 检查一次
			if err != nil {
				log.Errorf("server shell run error: %s", err)
			}
		})
	}

	// 启动检测机器 ssh可连接性并依据配置发送钉钉告警通知
	if app.App.Config.WithSSHCheck.Enable {
		app.App.Config.WithSSHCheck.LivenessCache = cache.New(cache.NoExpiration, cache.NoExpiration)
		log.Infof("with ssh check,5min check once")
		c.AddFunc("0 */5 * * * *", func() {
			log.Infof("run ssh check")
			core.ServerLiveness(app.App.Config.WithSSHCheck.Alert.RobotToken)
		})
	}

	c.Start()
	select {}
}
