package wecom

type Wecom struct {
	CorpId     string   `json:"corpId"`
	CorpSecret string   `json:"corpSecret"`
	AgentId    int      `json:"agentId"`
	Receivers  []string `json:"receivers"`
}
