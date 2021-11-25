package gh

import "time"

type Config struct {
	Token    string        `json:"token"`
	Repo     string        `json:"repo"`
	Interval time.Duration `json:"interval,default=1m"`
	Verbose  bool          `json:"verbose,default=false"`
}
