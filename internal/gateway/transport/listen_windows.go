//go:build windows

package transport

import (
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
)

// Listen 在 Windows 系统上启动 Named Pipe 监听。
func Listen(address string) (net.Listener, error) {
	listener, err := winio.ListenPipe(address, nil)
	if err != nil {
		return nil, fmt.Errorf("gateway: listen named pipe: %w", err)
	}
	return newCleanupListener(listener, nil), nil
}
