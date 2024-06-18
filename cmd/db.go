/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xops-infra/jms/app"
	appConfig "github.com/xops-infra/jms/model"
	"github.com/xops-infra/jms/utils"
	"github.com/xops-infra/noop/log"
)

// dbCmd represents the db command
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
		_app := app.NewSshdApplication(debug, logDir, rootCmd.Version)
		_app.WithDB()
		switch args[0] {
		case "upgrade":
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
}
