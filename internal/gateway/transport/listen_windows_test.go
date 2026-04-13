//go:build windows

package transport

import (
	"fmt"
	"testing"
	"time"

	"github.com/Microsoft/go-winio"
)

func TestListenNamedPipeAcceptsConnection(t *testing.T) {
	t.Parallel()

	pipePath := fmt.Sprintf(`\\.\pipe\neocode-gateway-test-%d`, time.Now().UnixNano())
	listener, err := Listen(pipePath)
	if err != nil {
		t.Fatalf("listen named pipe: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	acceptDone := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			acceptDone <- acceptErr
			return
		}
		_ = conn.Close()
		acceptDone <- nil
	}()

	timeout := 2 * time.Second
	conn, err := winio.DialPipe(pipePath, &timeout)
	if err != nil {
		t.Fatalf("dial named pipe: %v", err)
	}
	_ = conn.Close()

	select {
	case acceptErr := <-acceptDone:
		if acceptErr != nil {
			t.Fatalf("accept connection: %v", acceptErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("accept timed out")
	}
}
