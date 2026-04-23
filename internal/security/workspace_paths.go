package security

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// ResolveWorkspacePath 按 workspace sandbox 的既有语义解析并校验工作区路径。
func ResolveWorkspacePath(root string, target string) (string, string, error) {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return "", "", errors.New("security: workspace root is empty")
	}

	absoluteRoot, err := filepath.Abs(trimmedRoot)
	if err != nil {
		return "", "", fmt.Errorf("security: resolve workspace root: %w", err)
	}

	canonicalRoot, _, err := resolveCanonicalWorkspaceRoot(cleanedPathKey(absoluteRoot))
	if err != nil {
		return "", "", err
	}

	absoluteTarget, err := ResolveWorkspacePathFromRoot(canonicalRoot, target)
	if err != nil {
		return "", "", err
	}
	return canonicalRoot, absoluteTarget, nil
}

// ResolveWorkspacePathFromRoot 在已知 canonical workspace root 的前提下解析并校验目标路径。
func ResolveWorkspacePathFromRoot(root string, target string) (string, error) {
	absoluteTarget, err := absoluteWorkspaceTarget(root, target)
	if err != nil {
		return "", err
	}
	if !isWithinWorkspace(root, absoluteTarget) {
		return "", fmt.Errorf("security: path %q escapes workspace root", target)
	}
	if _, err := ensureNoSymlinkEscape(root, absoluteTarget, target); err != nil {
		return "", err
	}
	return absoluteTarget, nil
}

// ResolveWorkspaceWalkPathFromRoot 在已知 canonical workspace root 的前提下，
// 为遍历热路径做轻量校验：普通文件只做 containment，符号链接条目再回落到完整校验。
func ResolveWorkspaceWalkPathFromRoot(root string, target string, entry fs.DirEntry) (string, error) {
	absoluteTarget, err := absoluteWorkspaceTarget(root, target)
	if err != nil {
		return "", err
	}
	if !isWithinWorkspace(root, absoluteTarget) {
		return "", fmt.Errorf("security: path %q escapes workspace root", target)
	}
	if entry != nil && entry.Type().IsRegular() {
		return absoluteTarget, nil
	}
	if _, err := ensureNoSymlinkEscape(root, absoluteTarget, target); err != nil {
		return "", err
	}
	return absoluteTarget, nil
}
