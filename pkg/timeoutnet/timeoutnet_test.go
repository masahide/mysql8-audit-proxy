package timeoutnet

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestTimeoutReaderAndWriter(t *testing.T) {
	// net.Pipeを使用して双方向のパイプを作成し、それをnet.Connのモックとして使用します。
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	// エコーサーバーの代わりに、双方向パイプを使用してデータをエコーするゴルーチンを作成します。
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn2.Read(buf)
			if err != nil {
				return
			}
			time.Sleep(500 * time.Millisecond) // ここで遅延を追加
			_, err = conn2.Write(buf[:n])
			if err != nil {
				return
			}
		}
	}()

	// タイムアウトを設定し、キャンセルが正しく動作するかどうかをテストします。
	t.Run("TestCancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		timeoutReader := &TimeoutReader{
			Conn:    conn1,
			Timeout: 1 * time.Second,
			Ctx:     ctx,
		}
		timeoutWriter := &TimeoutWriter{
			Conn:    conn1,
			Timeout: 1 * time.Second,
			Ctx:     ctx,
		}

		data := bytes.Repeat([]byte("Hello, world!"), 1000)
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()

		//t.Logf("data len=%d", len(data))
		_, err := timeoutWriter.Write(data)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}

		var out bytes.Buffer
		_, err = io.Copy(&out, timeoutReader)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	})
	// タイムアウトが正しく動作するかどうかをテストします。
	t.Run("TestTimeout", func(t *testing.T) {
		timeoutReader := &TimeoutReader{
			Conn:    conn1,
			Timeout: 1 * time.Second,
			Ctx:     context.Background(),
		}
		timeoutWriter := &TimeoutWriter{
			Conn:    conn1,
			Timeout: 1 * time.Second,
			Ctx:     context.Background(),
		}

		data := strings.Repeat("Timeout test", 10000)
		_, err := timeoutWriter.Write([]byte(data))
		if err == nil || !strings.Contains(err.Error(), "timeout") {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
		out := make([]byte, 100)
		_, err = timeoutReader.Read(out)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(out) == 0 {
			t.Errorf("Expected non-empty output buffer, got empty")
		}
	})
}
