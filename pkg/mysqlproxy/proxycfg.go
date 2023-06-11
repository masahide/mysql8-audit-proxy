package mysqlproxy

import "time"

type ProxyCfg struct {
	Net             string
	Addr            string
	ProxyListenAddr string `default:":3307` // "localhost:3307"
	ProxyListentNet string `default:"tcp"`  // tcp or unix
	ConTimeout      time.Duration

	LogFileName     string        `default:"mysql-audit.%Y%m%d%H.log"`
	EncodeType      string        `default:"binary"`
	LogGzip         bool          `default:"true"`
	RotateTime      time.Duration `default:"1h"`
	BufSizeMB       string        `default:"32"`
	QueueSize       int           `default:"200"`
	BufferFlushTime time.Duration `default:"1s"`
}

type ProxyUser struct {
	Username string
	Password string
}
