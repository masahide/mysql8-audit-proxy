package mysqlproxy

import "time"

type ProxyCfg struct {
	ProxyListenAddr string
	ProxyListenPort string // 3306
	ProxyListentNet string // tcp or unix
	TargetAddr      string
	TargetPort      string // 3306
	TargetNet       string // tcp or unix
	TargetUser      string
	TargetPass      string
	TargetDB        string
	ProxyUsers      []ProxyUser
	ConTimeout      time.Duration
}

type ProxyUser struct {
	Username string
	Password string
}
