package mysqlproxy

import (
	"time"
)

type ProxyCfg struct {
	ProxyListenAddr string        `envconfig:"PROXY_LISTEN_ADDR" default:":3307"` // "localhost:3307"
	ProxyListenNet  string        `envconfig:"PROXY_LISTEN_NET" default:"tcp"`    // tcp or unix
	ConTimeout      time.Duration `envconfig:"CON_TIMEOUT" default:"300s"`
	LogFileName     string        `envconfig:"LOG_FILE_NAME" default:"mysql8-audit.%Y%m%d%H.log.gz"`
	RotateTime      time.Duration `envconfig:"ROTATE_TIME" default:"1h"`
	AdminUser       string        `envconfig:"ADMIN_USER" default:"admin"`
	ConfigDir       string        `envconfig:"PROXY_CONFIG_DIR"`
	Debug           bool          `envconfig:"DEBUG" default:"false"`
}

type ProxyUser struct {
	Username string
	Password string
}
