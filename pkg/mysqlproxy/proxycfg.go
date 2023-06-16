package mysqlproxy

import (
	"time"
)

type ProxyCfg struct {
	ProxyListenAddr string        `default:":3307"` // "localhost:3307"
	ProxyListentNet string        `default:"tcp"`   // tcp or unix
	ConTimeout      time.Duration `default:"300s"`

	LogFileName     string        `default:"mysql-audit.%Y%m%d%H.log.gz"`
	EncodeType      string        `default:"binary"`
	LogGzip         bool          `default:"true"`
	RotateTime      time.Duration `default:"1h"`
	BufSizeMB       string        `default:"32"`
	QueueSize       int           `default:"200"`
	BufferFlushTime time.Duration `default:"1s"`
	AdminUser       string        `default:"admin"`
	Debug           bool          `default:"false"`
}

type ProxyUser struct {
	Username string
	Password string
}
