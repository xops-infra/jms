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
	"github.com/spf13/cobra"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/api"
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

		if !app.App.Config.WithDB.Enable {
			panic("请配置 withDB")
		}

		if app.App.Config.WithDingtalk.Enable {
			log.Infof("enable dingtalk")
			_app.WithDingTalk()
		}
		_app.WithDB()

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
