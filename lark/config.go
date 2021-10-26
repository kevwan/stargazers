package lark

type Lark struct {
	AppId         string `json:"appId"`
	AppSecret     string `json:"appSecret"`
	Receiver      string `json:"receiver,optional"`
	ReceiverEmail string `json:"receiver_email,optional=!receiver"`
}
