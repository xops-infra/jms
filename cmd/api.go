/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/xops-infra/noop/log"

	"github.com/google/gops/agent"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/api"
	"github.com/xops-infra/jms/model"
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

		model.InitConfig(config)

		// init app
		_app := app.NewApplication(debug, logDir, rootCmd.Version, config)

		if app.App.Config.WithDB.Enable {
			log.Infof("enable db without automigrate")
			_app.WithDB(false)
		}

		log.Infof("api server start on port: %d", apiPort)
		if !debug {
			gin.SetMode(gin.ReleaseMode)
		}
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
