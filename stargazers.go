package main

import (
	"flag"
	"log"

	"stargazers/gh"
	"stargazers/lark"
	"stargazers/lark/webhook"
	"stargazers/slack"

	"github.com/tal-tech/go-zero/core/conf"
	"github.com/tal-tech/go-zero/core/logx"
)

var configFile = flag.String("f", "config.yaml", "the config file")

type Config struct {
	gh.Config
	Lark  *lark.Lark   `json:"lark,optional"`
	Slack *slack.Slack `json:"slack,optional"`
}

func getSender(c Config) func(string) error {
	if c.Lark != nil {
		app := lark.NewApp(c.Lark.AppId, c.Lark.AppSecret)
		if len(c.Lark.Receiver) > 0 || len(c.Lark.ReceiverEmail) > 0 {
			return func(message string) error {
				return app.Send(
					c.Lark.Receiver,
					c.Lark.ReceiverEmail,
					message,
				)
			}
		} else if len(c.Lark.WebhookUrl) > 0 {
			return func(message string) error {
				return webhook.Send(c.Lark.WebhookUrl, message)
			}
		}
	}

	if c.Slack != nil {
		return func(message string) error {
			return slack.Send(
				c.Slack.Token,
				c.Slack.Channel,
				message,
			)
		}
	}

	return nil
}

func main() {
	flag.Parse()

	var c Config
	conf.MustLoad(*configFile, &c)
	sender := getSender(c)
	if sender == nil {
		log.Fatal("Set either lark, webhook or slack to receive notifications.")
	}
	mon := gh.NewMonitor(c.Config, sender)
	logx.Must(mon.Start())
}
