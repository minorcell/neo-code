package session

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

const projectsDirName = "projects"

// sessionDirectory 负责根据工作区根目录计算会话分桶目录。
func sessionDirectory(baseDir string, workspaceRoot string) string {
	return filepath.Join(baseDir, projectsDirName, HashWorkspaceRoot(workspaceRoot), sessionsDirName)
}

// HashWorkspaceRoot 为规范化后的工作区根目录生成稳定哈希，供 session 和 memo 等包共享。
func HashWorkspaceRoot(workspaceRoot string) string {
	key := WorkspacePathKey(workspaceRoot)
	if key == "" {
		key = "unknown"
	}
	sum := sha1.Sum([]byte(key))
	return hex.EncodeToString(sum[:8])
}

// WorkspacePathKey 生成工作区路径的稳定比较键，Windows 下兼容大小写不敏感。
func WorkspacePathKey(workspaceRoot string) string {
	normalized := NormalizeWorkspaceRoot(workspaceRoot)
	if normalized == "" {
		return ""
	}
	if goruntime.GOOS == "windows" {
		return strings.ToLower(normalized)
	}
	return normalized
}

// NormalizeWorkspaceRoot 将工作区根目录规范化为绝对清洗路径。
func NormalizeWorkspaceRoot(workspaceRoot string) string {
	trimmed := strings.TrimSpace(workspaceRoot)
	if trimmed == "" {
		return ""
	}

	absolute, err := filepath.Abs(trimmed)
	if err == nil {
		trimmed = absolute
	}
	return filepath.Clean(trimmed)
}
