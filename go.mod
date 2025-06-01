module github.com/masahide/mysql8-audit-proxy

go 1.24.3

require (
	github.com/dzeromsk/debpack v0.0.0-20190912160929-4b3d7b5dd69b
	github.com/go-mysql-org/go-mysql v1.12.0
	github.com/google/go-cmp v0.5.9
	github.com/google/rpmpack v0.7.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pingcap/tidb/parser v0.0.0-20231013125129-93a834a6bf8d
)

replace github.com/go-mysql-org/go-mysql => github.com/masahide/go-mysql v0.0.0-20250531153824-537447821909

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/blakesmith/ar v0.0.0-20190502131153-809d4375e1fb // indirect
	github.com/cavaliergopher/cpio v1.0.1 // indirect
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/pingcap/errors v0.11.5-0.20250318082626-8f80e5cb09ec // indirect
	github.com/pingcap/failpoint v0.0.0-20240528011301-b51a646c7c86 // indirect
	github.com/pingcap/log v1.1.1-0.20241212030209-7e3ff8601a2a // indirect
	github.com/pingcap/tidb/pkg/parser v0.0.0-20250531022214-e7b038b99132 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/ulikunitz/xz v0.5.12 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
