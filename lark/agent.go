package lark

import (
	"encoding/json"

	"github.com/fastwego/feishu"
	"github.com/fastwego/feishu/apis/message"
)

const messageType = "text"

type (
	App struct {
		app *feishu.App
	}

	Message struct {
		UserId  string  `json:"user_id"`
		Email   string  `json:"email"`
		MsgType string  `json:"msg_type"`
		Content content `json:"content"`
	}

	content struct {
		Text string `json:"text"`
	}
)

func NewApp(appid, secret string) *App {
	return &App{
		app: feishu.NewApp(feishu.AppConfig{
			AppId:     appid,
			AppSecret: secret,
		}),
	}
}

func (app *App) Send(receiver, receiverEmail, text string) error {
	payload, err := json.Marshal(Message{
		UserId:  receiver,
		Email:   receiverEmail,
		MsgType: messageType,
		Content: content{
			Text: text,
		},
	})
	if err != nil {
		return err
	}

	_, err = message.Send(app.app, payload)
	return err
}
