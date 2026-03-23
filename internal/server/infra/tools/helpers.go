package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func requiredString(params map[string]interface{}, key string) (string, *ToolResult) {
	value, ok := params[key]
	if !ok {
		return "", &ToolResult{Success: false, Error: fmt.Sprintf("缺少必需参数: %s", key)}
	}
	str, ok := value.(string)
	if !ok || strings.TrimSpace(str) == "" {
		return "", &ToolResult{Success: false, Error: fmt.Sprintf("%s 必须是非空字符串", key)}
	}
	return str, nil
}

func optionalString(params map[string]interface{}, key, fallback string) (string, *ToolResult) {
	value, ok := params[key]
	if !ok {
		return fallback, nil
	}
	str, ok := value.(string)
	if !ok {
		return "", &ToolResult{Success: false, Error: fmt.Sprintf("%s 必须是字符串", key)}
	}
	if strings.TrimSpace(str) == "" {
		return fallback, nil
	}
	return str, nil
}

func optionalInt(params map[string]interface{}, key string, fallback int) (int, *ToolResult) {
	value, ok := params[key]
	if !ok {
		return fallback, nil
	}
	switch v := value.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0, &ToolResult{Success: false, Error: fmt.Sprintf("%s 必须是数字", key)}
		}
		return parsed, nil
	default:
		return 0, &ToolResult{Success: false, Error: fmt.Sprintf("%s 必须是数字", key)}
	}
}

func optionalBool(params map[string]interface{}, key string, fallback bool) (bool, *ToolResult) {
	value, ok := params[key]
	if !ok {
		return fallback, nil
	}
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		default:
			return false, &ToolResult{Success: false, Error: fmt.Sprintf("%s 必须是布尔值", key)}
		}
	default:
		return false, &ToolResult{Success: false, Error: fmt.Sprintf("%s 必须是布尔值", key)}
	}
}

func workspaceRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func resolveWorkspacePath(path string) (string, error) {
	root, err := filepath.Abs(workspaceRoot())
	if err != nil {
		return "", err
	}
	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	if candidate != root && !strings.HasPrefix(candidate, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("路径超出工作区: %s", path)
	}
	return candidate, nil
}

func ensureWorkspacePath(path string) (string, *ToolResult) {
	resolved, err := resolveWorkspacePath(path)
	if err != nil {
		return "", &ToolResult{Success: false, Error: fmt.Sprintf("无效路径: %v", err)}
	}
	return resolved, nil
}

// AtomicWrite 以原子方式将内容写入文件，确保文件完整性。
func AtomicWrite(filePath string, content []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 1. 在同目录下创建临时文件
	tmpFile, err := os.CreateTemp(dir, "neocode-tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath) // 正常 Rename 后 Remove 会失败，属正常行为
	}()

	// 2. 写入内容
	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 3. 强制刷盘，确保操作系统已将数据写入物理存储
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("刷盘失败: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}

	// 4. 原子重命名，替换目标文件
	// 在 Unix-like 和 Windows (同一卷内) 这通常是原子的
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("原子替换失败: %w", err)
	}

	return nil
}

func NormalizeParams(params map[string]interface{}) map[string]interface{} {
	if params == nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{}, len(params))
	for key, value := range params {
		camelKey := snakeToCamel(strings.TrimSpace(key))
		switch typed := value.(type) {
		case map[string]interface{}:
			result[camelKey] = NormalizeParams(typed)
		default:
			result[camelKey] = value
		}
	}
	return result
}

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) <= 1 {
		return s
	}
	out := parts[0]
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	return out
}
