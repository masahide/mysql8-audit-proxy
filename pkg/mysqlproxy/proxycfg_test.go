package mysqlproxy

import (
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
)

func TestProxyCfgUsesReadmeEnvironmentVariables(t *testing.T) {
	t.Setenv("PROXY_LISTEN_ADDR", "127.0.0.1:4307")
	t.Setenv("PROXY_LISTEN_NET", "unix")
	t.Setenv("CON_TIMEOUT", "45s")
	t.Setenv("LOG_FILE_NAME", "custom-audit.log.gz")
	t.Setenv("ROTATE_TIME", "30m")
	t.Setenv("ADMIN_USER", "root-admin")
	t.Setenv("PROXY_CONFIG_DIR", "/tmp/mysql8-audit-proxy-config")
	t.Setenv("DEBUG", "true")

	var cfg ProxyCfg
	if err := envconfig.Process("", &cfg); err != nil {
		t.Fatalf("envconfig.Process() error = %v", err)
	}

	if cfg.ProxyListenAddr != "127.0.0.1:4307" {
		t.Fatalf("ProxyListenAddr = %q, want %q", cfg.ProxyListenAddr, "127.0.0.1:4307")
	}
	if cfg.ProxyListenNet != "unix" {
		t.Fatalf("ProxyListenNet = %q, want %q", cfg.ProxyListenNet, "unix")
	}
	if cfg.ConTimeout != 45*time.Second {
		t.Fatalf("ConTimeout = %v, want %v", cfg.ConTimeout, 45*time.Second)
	}
	if cfg.LogFileName != "custom-audit.log.gz" {
		t.Fatalf("LogFileName = %q, want %q", cfg.LogFileName, "custom-audit.log.gz")
	}
	if cfg.RotateTime != 30*time.Minute {
		t.Fatalf("RotateTime = %v, want %v", cfg.RotateTime, 30*time.Minute)
	}
	if cfg.AdminUser != "root-admin" {
		t.Fatalf("AdminUser = %q, want %q", cfg.AdminUser, "root-admin")
	}
	if cfg.ConfigDir != "/tmp/mysql8-audit-proxy-config" {
		t.Fatalf("ConfigDir = %q, want %q", cfg.ConfigDir, "/tmp/mysql8-audit-proxy-config")
	}
	if !cfg.Debug {
		t.Fatal("Debug = false, want true")
	}
}
