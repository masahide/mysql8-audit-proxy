package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy"
	proxylog "github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/log"
	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/sendpacket"
	"github.com/masahide/mysql8-audit-proxy/pkg/serverconfig"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	showVer = flag.Bool("version", false, "Show version")
)

func main() {
	flag.Parse()
	if *showVer {
		fmt.Printf("version: %v\ncommit: %v\nbuilt_at: %v\n", version, commit, date)
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	confDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	svConfMng := serverconfig.NewManager(confDir)
	proxyConf := &mysqlproxy.ProxyCfg{}
	if err := envconfig.Process("", proxyConf); err != nil {
		log.Fatal(err)
	}
	if proxyConf.Debug {
		log.Printf("serverConfig %s", svConfMng.PrintPathInfo())
		log.Printf("proxyConfig:\n%s\n", dumpJSON(proxyConf))
	}
	q := make(chan *sendpacket.SendPacket, 1000)
	logHandler, err := proxylog.NewAuditLogWriter(q, proxyConf.LogFileName, proxyConf.RotateTime, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	p := &mysqlproxy.ProxySrv{
		AuditLogWriter: logHandler,
		SvConfMng:      svConfMng,
		Config:         proxyConf,
	}

	pctx, cancel := context.WithCancel(context.Background())
	ctx, stop := signal.NotifyContext(pctx, os.Interrupt)
	defer stop()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := logHandler.LogWriteWorker(ctx)
		log.Println("logWriteWorker stopped")
		if err != nil && err != context.Canceled {
			log.Printf("Proxy error: %v", err)
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		err := p.Start(ctx)
		log.Println("proxy stopped")
		if err != nil && err != context.Canceled {
			log.Printf("Proxy error: %v", err)
		}
		logHandler.CloseChannel()
		cancel()
		wg.Done()
	}()
	wg.Wait()
}

func dumpJSON(v any) string {
	s, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(s)
}
