package main

import (
	"flag"
	"log"
	"time"

	"stargazers/feishu"
	"stargazers/gh"

	"github.com/tal-tech/go-zero/core/conf"
)

var configFile = flag.String("f", "config.yaml", "the config file")

type (
	Feishu struct {
		AppId         string `json:"appId"`
		AppSecret     string `json:"appSecret"`
		Receiver      string `json:"receiver,optional"`
		ReceiverEmail string `json:"receiver_email,optional=!receiver"`
	}

	Config struct {
		Token    string        `json:"token"`
		Repo     string        `json:"repo"`
		Interval time.Duration `json:"interval,default=1m"`
		Feishu   Feishu        `json:"feishu"`
	}
)

func main() {
	flag.Parse()

	var c Config
	conf.MustLoad(*configFile, &c)

	mon := gh.NewMonitor(c.Repo, c.Token, c.Interval, func(text string) error {
		return feishu.Send(c.Feishu.AppId, c.Feishu.AppSecret, c.Feishu.Receiver, c.Feishu.ReceiverEmail, text)
	})
	log.Fatal(mon.Start())
}
