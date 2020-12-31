package feishu

import (
	"encoding/json"

	"github.com/fastwego/feishu"
	"github.com/fastwego/feishu/apis/message"
)

const messageType = "text"

var App *feishu.App

type (
	content struct {
		Text string `json:"text"`
	}

	Message struct {
		Email  string   `json:"email"`
		MsgType string  `json:"msg_type"`
		Content content `json:"content"`
	}
)

func Send(app, secret, receiver, text string) error {
	App = feishu.NewApp(feishu.AppConfig{
		AppId:     app,
		AppSecret: secret,
	})
	payload, err := json.Marshal(Message{
		Email:  receiver,
		MsgType: messageType,
		Content: content{
			Text: text,
		},
	})
	if err != nil {
		return err
	}

	_, err = message.Send(App, payload)
	return err
}
