/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import "github.com/xops-infra/jms/cmd"

var version string

// @title           cbs manager API
// @version         v1
// @termsOfService  http://swagger.io/terms/
// @host            localhost:8013
// @BasePath
func main() {
	cmd.Execute(version)
}
