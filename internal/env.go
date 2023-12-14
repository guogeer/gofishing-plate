package internal

import (
	"github.com/guogeer/quasar/config"
)

type DataSource struct {
	User         string `xml:"User"`
	Password     string `xml:"Password"`
	Addr         string `xml:"Address"`
	Name         string `xml:"Name"`
	MaxIdleConns int
	MaxOpenConns int
}

type Env struct {
	Version          int
	Environment      string
	DataSource       DataSource
	ManageDataSource DataSource
	SlaveDataSource  DataSource
	ProductName      string
	ProductKey       string
	ResourceURL      string
	ResourcePath     string
	PrereleaseURL    string
	ApplePayPassword string

	GoogleAPIs struct {
		ServiceAccount string
	}
	Pubsub struct {
		ServiceAccount     string
		ProjectId          string
		SubscriptionId     string
		TestSubscriptionId string
	}
	Thinkingdata struct {
		URL   string
		AppId string
	}
	Mail struct {
		From         string
		To           string
		SMTPHost     string
		SMTPPort     int
		SMTPUser     string
		SMTPPassword string
	}

	FacebookInstallRefererKey string

	config.Env
}

func (env *Env) IsTest() bool {
	return env.Environment == "test"
}

var defaultConfig Env

func init() {
	config.LoadFile(config.Config().Path(), &defaultConfig)
}

func Config() *Env {
	return &defaultConfig
}

func GetClientEnv() int {
	switch Config().Environment {
	case "test":
		return 1
	case "release":
		return 2
	case "dev":
		return 3
	case "prerelease":
		return 4
	}
	return 0
}
