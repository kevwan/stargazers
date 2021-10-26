package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tal-tech/go-zero/core/jsonx"
	"github.com/tal-tech/go-zero/core/logx"
)

const slackPostMessageUrl = "https://slack.com/api/chat.postMessage"

type (
	Request struct {
		Channel string `json:"channel"`
		Text    string `json:"text"`
	}

	Response struct {
		OK    bool   `json:"ok"`
		Error string `json:",optional"`
	}
)

func Send(token, channel, message string) error {
	slackMsg := Request{
		Channel: channel,
		Text:    message,
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(slackMsg); err != nil {
		return err
	}

	r, err := http.NewRequest(http.MethodPost, slackPostMessageUrl, &buf)
	if err != nil {
		return err
	}

	r.Header.Set("Authorization", "Bearer "+token)
	resp, err := new(http.Client).Do(r)
	if err != nil {
		return err
	}

	var rsp Response
	if err := jsonx.UnmarshalFromReader(resp.Body, &rsp); err != nil {
		return err
	}

	if !rsp.OK {
		return errors.New(rsp.Error)
	}

	logx.Info("sent")
	return nil
}
