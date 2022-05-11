package wecom

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"stargazers/sender"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zeromicro/go-zero/rest/httpc"
)

const (
	messageType     = "text"
	refreshTokenUrl = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
)

type (
	accessToken struct {
		Token  string
		Expire time.Time
	}

	app struct {
		c     *Wecom
		token accessToken
	}

	textBody struct {
		Content string `json:"content"`
	}

	request struct {
		AccessToken string   `form:"access_token"`
		AgentID     int      `json:"agentid"`
		MsgType     string   `json:"msgtype,default=text"`
		ToUser      string   `json:"touser"`
		ToParty     string   `json:"toparty"`
		Text        textBody `json:"text"`
	}

	response struct {
		Code int    `json:"errcode"`
		Msg  string `json:"errmsg"`
	}

	wechatTokenResp struct {
		Code   int    `json:"errcode"`
		Msg    string `json:"errmsg"`
		Token  string `json:"access_token"`
		Expire int    `json:"expires_in"`
	}
)

func NewSender(c *Wecom) sender.Sender {
	return &app{c: c}
}

func (a *app) Send(text string) error {
	toUsers := strings.Join(a.c.Receivers, "|")
	token, err := a.getToken()
	if err != nil {
		return err
	}

	req := request{
		AccessToken: token,
		AgentID:     a.c.AgentId,
		MsgType:     messageType,
		ToUser:      toUsers,
		Text: textBody{
			Content: text,
		},
	}
	resp, err := httpc.Do(context.Background(), http.MethodPost, "https://qyapi.weixin.qq.com/cgi-bin/message/send", req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	var rsp response
	if err := httpc.Parse(resp, &rsp); err != nil {
		return err
	}
	if rsp.Code != 0 {
		return errors.New(rsp.Msg)
	}

	return nil
}

func (a *app) getToken() (string, error) {
	if time.Since(a.token.Expire) <= time.Duration(-30)*time.Second {
		return a.token.Token, nil
	}

	// refetch access token
	resp, err := httpc.Do(context.Background(), http.MethodGet, refreshTokenUrl, struct {
		CorpId string `form:"corpid"`
		Secret string `form:"corpsecret"`
	}{
		CorpId: a.c.CorpId,
		Secret: a.c.CorpSecret,
	})
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	var result wechatTokenResp
	if err := httpc.Parse(resp, &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		logx.Error(result)
		return "", fmt.Errorf("errcode: %d", result.Code)
	}

	var exp = time.Now().Add(time.Duration(result.Expire) * time.Second)
	a.token = accessToken{
		Token:  result.Token,
		Expire: exp,
	}

	return a.token.Token, nil
}
