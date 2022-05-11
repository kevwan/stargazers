package slack

import (
	"context"
	"errors"
	"net/http"

	"stargazers/sender"

	"github.com/zeromicro/go-zero/rest/httpc"
)

const slackPostMessageUrl = "https://slack.com/api/chat.postMessage"

type (
	app struct {
		c *Slack
	}

	request struct {
		Channel       string `json:"channel"`
		Text          string `json:"text"`
		Authorization string `header:"Authorization"`
	}

	response struct {
		OK    bool   `json:"ok"`
		Error string `json:",optional"`
	}
)

func NewSender(c *Slack) sender.Sender {
	return &app{c: c}
}

func (a *app) Send(message string) error {
	req := request{
		Channel:       a.c.Channel,
		Text:          message,
		Authorization: "Bearer " + a.c.Token,
	}
	resp, err := httpc.Do(context.Background(), http.MethodPost, slackPostMessageUrl, req)
	if err != nil {
		return err
	}

	var rsp response
	if err := httpc.Parse(resp, &rsp); err != nil {
		return err
	}

	if !rsp.OK {
		return errors.New(rsp.Error)
	}

	return nil
}
