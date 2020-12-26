package main

import (
	"flag"
	"log"

	"stargazers/gh"

	"github.com/tal-tech/go-zero/core/conf"
)

var configFile = flag.String("f", "config.yaml", "the config file")

type (
	feishu struct {
		AppId     string `json:"appId"`
		AppSecret string `json:"appSecret"`
		Receiver  string `json:"receiver"`
	}

	Config struct {
		Token  string `json:"token"`
		Repo   string `json:"repo"`
		Feishu feishu `json:"feishu"`
	}
)

func main() {
	flag.Parse()

	var c Config
	conf.MustLoad(*configFile, &c)

	mon := gh.NewMonitor(c.Repo, c.Token, c.Feishu.AppId, c.Feishu.AppSecret, c.Feishu.Receiver)
	log.Fatal(mon.Start())
}
