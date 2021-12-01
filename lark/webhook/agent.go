package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tal-tech/go-zero/core/jsonx"
	"github.com/tal-tech/go-zero/core/logx"
)

type Response struct {
	StatusCode    int    `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`
}

func Send(url, message string) error {
	slackMsg := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]interface{}{
			"text": message,
		},
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(slackMsg); err != nil {
		return err
	}

	r, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return err
	}

	resp, err := new(http.Client).Do(r)
	if err != nil {
		return err
	}

	var rsp Response
	if err := jsonx.UnmarshalFromReader(resp.Body, &rsp); err != nil {
		return err
	}

	if rsp.StatusCode != 0 {
		return errors.New(rsp.StatusMessage)
	}

	logx.Info("sent")
	return nil
}
