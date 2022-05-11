package main

import (
	"flag"
	"log"

	"stargazers/gh"
	"stargazers/lark"
	"stargazers/sender"
	"stargazers/slack"
	"stargazers/trending"
	"stargazers/wecom"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "config.yaml", "the config file")

type Config struct {
	gh.Config
	Trending trending.Trending `json:"trending,optional"`
	Lark     *lark.Lark        `json:"lark,optional"`
	Slack    *slack.Slack      `json:"slack,optional"`
	Wecom    *wecom.Wecom      `json:"wecom,optional"`
}

func getSender(c Config) sender.Sender {
	if c.Lark != nil {
		return lark.NewSender(c.Lark)
	}

	if c.Slack != nil {
		return slack.NewSender(c.Slack)
	}

	if c.Wecom != nil {
		return wecom.NewSender(c.Wecom)
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

	group := service.NewServiceGroup()
	group.Add(service.WithStarter(gh.NewMonitor(c.Config, sender)))
	group.Add(service.WithStarter(trending.NewMonitor(c.Repo, c.Trending, sender)))
	group.Start()
}
