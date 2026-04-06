package openai

import (
	"bufio"
	"errors"
	"io"

	"neo-code/internal/provider"
)

// 单行与总量上限，防止恶意或异常数据导致内存无限增长。
const (
	maxSSELineSize     = 256 * 1024 // L1: 单行 256KB
	maxStreamTotalSize = 10 << 20   // L3: 总量 10MB
)

// boundedSSEReader 对 bufio.Reader 包装两级有界检查：
//   - L1: 每次读取的行不超过 maxSSELineSize
//   - L3: 累计读取字节数不超过 maxStreamTotalSize
//
// 纯同步设计，无 goroutine/channel，适用于 SSE 顺序消费场景。
type boundedSSEReader struct {
	reader    *bufio.Reader
	totalRead int64
}

// newBoundedSSEReader 创建有界 SSE 行读取器。
func newBoundedSSEReader(r io.Reader) *boundedSSEReader {
	return &boundedSSEReader{
		reader: bufio.NewReader(r),
	}
}

// ReadLine 读取一行（以 \n 分隔），同时执行 L1 和 L3 检查。
// 返回去除尾部 \r\n 的行内容；遇到 io.EOF 时返回空字符串和 nil。
func (r *boundedSSEReader) ReadLine() (string, error) {
	line, err := r.reader.ReadString('\n')

	// L3: 总量检查（在 L1 之后、返回前统一判断）
	r.totalRead += int64(len(line))
	if r.totalRead > maxStreamTotalSize {
		return "", provider.ErrStreamTooLarge
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	// L1: 单行长度检查（不含末尾 \n）
	rawLen := len(line)
	if rawLen > 0 && line[rawLen-1] == '\n' {
		rawLen--
	}
	if rawLen > maxSSELineSize {
		return "", provider.ErrLineTooLong
	}

	// 去除尾部 \r\n
	return trimLineEnding(line), err
}

// trimLineEnding 移除行尾的 \r\n 或 \n。
func trimLineEnding(line string) string {
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	return line
}
