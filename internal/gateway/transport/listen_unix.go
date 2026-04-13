//go:build !windows

package transport

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// Listen 在 Unix 系统上启动 UDS 监听并在关闭时清理 socket 文件。
func Listen(address string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(address), 0o755); err != nil {
		return nil, fmt.Errorf("gateway: create socket dir: %w", err)
	}

	if err := removeStaleUnixSocket(address); err != nil {
		return nil, err
	}

	listener, err := net.Listen("unix", address)
	if err != nil {
		return nil, fmt.Errorf("gateway: listen unix socket: %w", err)
	}

	return newCleanupListener(listener, func() error {
		if err := os.Remove(address); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("gateway: remove unix socket: %w", err)
		}
		return nil
	}), nil
}

// removeStaleUnixSocket 清理历史残留的 socket 文件，避免监听失败。
func removeStaleUnixSocket(address string) error {
	info, err := os.Lstat(address)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("gateway: stat unix socket path: %w", err)
	}

	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("gateway: unix socket path exists and is not socket: %s", address)
	}

	if err := os.Remove(address); err != nil {
		return fmt.Errorf("gateway: remove stale unix socket: %w", err)
	}

	return nil
}
