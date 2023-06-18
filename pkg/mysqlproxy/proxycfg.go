package mysqlproxy

import (
	"time"
)

type ProxyCfg struct {
	ProxyListenAddr string        `default:":3307"` // "localhost:3307"
	ProxyListentNet string        `default:"tcp"`   // tcp or unix
	ConTimeout      time.Duration `default:"300s"`
	LogFileName     string        `default:"mysql-audit.%Y%m%d%H.log.gz"`
	RotateTime      time.Duration `default:"1h"`
	AdminUser       string        `default:"admin"`
	Debug           bool          `default:"false"`
}

type ProxyUser struct {
	Username string
	Password string
}
