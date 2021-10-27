package main

import (
	"flag"
	"log"
	"time"

	"stargazers/gh"
	"stargazers/lark"
	"stargazers/slack"

	"github.com/tal-tech/go-zero/core/conf"
	"github.com/tal-tech/go-zero/core/logx"
)

var configFile = flag.String("f", "config.yaml", "the config file")

type (
	Config struct {
		Token    string        `json:"token"`
		Repo     string        `json:"repo"`
		PageSize int           `json:"pageSize,default=100"`
		Interval time.Duration `json:"interval,default=1m"`
		Lark     *lark.Lark    `json:"lark,optional"`
		Slack    *slack.Slack  `json:"slack,optional"`
	}
)

func getSenders(c Config) []func(string) error {
	var senders []func(string) error

	if c.Lark != nil {
		senders = append(senders, func(message string) error {
			return lark.Send(
				c.Lark.AppId,
				c.Lark.AppSecret,
				c.Lark.Receiver,
				c.Lark.ReceiverEmail,
				message,
			)
		})
	}

	if c.Slack != nil {
		senders = append(senders, func(message string) error {
			return slack.Send(
				c.Slack.Token,
				c.Slack.Channel,
				message,
			)
		})
	}

	return senders
}

func main() {
	flag.Parse()

	var c Config
	conf.MustLoad(*configFile, &c)
	senders := getSenders(c)
	if len(senders) == 0 {
		log.Fatal("Set either Lark or Slack to receive notifications.")
	}

	mon := gh.NewMonitor(c.Repo, c.Token, c.PageSize, c.Interval, senders)
	logx.Must(mon.Start())
}
