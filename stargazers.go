package main

import (
	"flag"
	"time"

	"stargazers/feishu"
	"stargazers/gh"

	"github.com/tal-tech/go-zero/core/conf"
	"github.com/tal-tech/go-zero/core/logx"
)

var configFile = flag.String("f", "config.yaml", "the config file")

type Config struct {
	Token    string        `json:"token"`
	Repo     string        `json:"repo"`
	Interval time.Duration `json:"interval,default=1m"`
	Feishu   feishu.Feishu `json:"feishu,optional"`
}

func main() {
	flag.Parse()

	var c Config
	conf.MustLoad(*configFile, &c)

	mon := gh.NewMonitor(c.Repo, c.Token, c.Interval, func(text string) error {
		return feishu.Send(c.Feishu.AppId, c.Feishu.AppSecret, c.Feishu.Receiver, c.Feishu.ReceiverEmail, text)
	})
	logx.Must(mon.Start())
}
