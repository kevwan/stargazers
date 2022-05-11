package lark

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"stargazers/sender"

	"github.com/zeromicro/go-zero/rest/httpc"

	"github.com/fastwego/feishu"
	"github.com/fastwego/feishu/apis/message"
)

const messageType = "text"

type (
	app struct {
		app           *feishu.App
		receiver      string
		receiverEmail string
	}

	larkMessage struct {
		UserId  string  `json:"user_id"`
		Email   string  `json:"email"`
		MsgType string  `json:"msg_type"`
		Content content `json:"content"`
	}

	content struct {
		Text string `json:"text"`
	}

	textBody struct {
		Text string `json:"text"`
	}

	request struct {
		MsgType string   `json:"msg_type"`
		Content textBody `json:"content"`
	}

	response struct {
		StatusCode    int    `json:"StatusCode"`
		StatusMessage string `json:"StatusMessage"`
	}

	webhookApp struct {
		url string
	}
)

func NewSender(c *Lark) sender.Sender {
	if len(c.Receiver) > 0 || len(c.ReceiverEmail) > 0 {
		return newApp(c)
	}

	return newWebhook(c)
}

func newApp(c *Lark) *app {
	return &app{
		app: feishu.NewApp(feishu.AppConfig{
			AppId:     c.AppId,
			AppSecret: c.AppSecret,
		}),
		receiver:      c.Receiver,
		receiverEmail: c.ReceiverEmail,
	}
}

func (a *app) Send(text string) error {
	payload, err := json.Marshal(larkMessage{
		UserId:  a.receiver,
		Email:   a.receiverEmail,
		MsgType: messageType,
		Content: content{
			Text: text,
		},
	})
	if err != nil {
		return err
	}

	_, err = message.Send(a.app, payload)
	return err
}

func newWebhook(c *Lark) sender.Sender {
	return &webhookApp{
		url: c.WebhookUrl,
	}
}

func (a *webhookApp) Send(message string) error {
	req := request{
		MsgType: messageType,
		Content: textBody{
			Text: message,
		},
	}

	resp, err := httpc.Do(context.Background(), http.MethodPost, a.url, req)
	if err != nil {
		return err
	}

	var rsp response
	if err := httpc.Parse(resp, &rsp); err != nil {
		return err
	}

	if rsp.StatusCode != 0 {
		return errors.New(rsp.StatusMessage)
	}

	return nil
}
