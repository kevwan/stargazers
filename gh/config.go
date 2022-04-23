package gh

import "time"

type Config struct {
	Token       string        `json:"token"`
	Repo        string        `json:"repo"`
	Comparisons []string      `json:"comparisons,optional"`
	Interval    time.Duration `json:"interval,default=1m"`
	Expect      *struct {
		Date  string `json:"date"`
		Stars int    `json:"stars"`
	} `json:"expect,optional"`
	Verbose bool `json:"verbose,default=false"`
}
