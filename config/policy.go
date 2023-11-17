package config

import "github.com/xops-infra/multi-cloud-sdk/pkg/model"

type Policy struct {
	Name         string     `mapstructure:"name"`
	Enabled      bool       `mapstructure:"enabled"`
	Groups       []string   `mapstructure:"groups"`
	ServerFilter model.Tags `mapstructure:"serverFilter"`
	Action       Action     `mapstructure:"action"`
}

type Group struct {
	Name  string   `mapstructure:"name"`
	Users []string `mapstructure:"users"`
}

// action
type Action struct {
	Login    bool `mapstructure:"login"`
	Download bool `mapstructure:"download"`
	Upload   bool `mapstructure:"upload"`
}
