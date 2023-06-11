package log

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	fmtVersion = `{"format":"mysqlproxy-v1.00"}\n`
)

func checkFormat(r io.Reader) (string, error) {
	size := len(fmtVersion)
	b := make([]byte, size)
	n, err := r.Read(b)
	if err != nil {
		return "", err
	}
	if n != size {
		return "", fmt.Errorf("version not match size:%d", n)
	}
	return string(b), nil
}

// /path/to/mysql-audit.%Y%m%d%H.log
func time2Path(p string, t time.Time) string {
	p = strings.Replace(p, "%Y", fmt.Sprintf("%04d", t.Year()), -1)
	p = strings.Replace(p, "%y", fmt.Sprintf("%02d", t.Year()%100), -1)
	p = strings.Replace(p, "%m", fmt.Sprintf("%02d", t.Month()), -1)
	p = strings.Replace(p, "%d", fmt.Sprintf("%02d", t.Day()), -1)
	p = strings.Replace(p, "%H", fmt.Sprintf("%02d", t.Hour()), -1)
	p = strings.Replace(p, "%M", fmt.Sprintf("%02d", t.Minute()), -1)
	p = strings.Replace(p, "%S", fmt.Sprintf("%02d", t.Second()), -1)
	return p
}
