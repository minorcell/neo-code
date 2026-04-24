package verify

import (
	"fmt"
	"strings"
)

// metadataStringSlice 从 metadata 中解析字符串列表。
func metadataStringSlice(metadata map[string]any, key string) []string {
	if len(metadata) == 0 {
		return nil
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if normalized := strings.TrimSpace(item); normalized != "" {
				out = append(out, normalized)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if normalized := strings.TrimSpace(fmt.Sprintf("%v", item)); normalized != "" {
				out = append(out, normalized)
			}
		}
		return out
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	default:
		return nil
	}
}

// metadataStringMapSlice 从 metadata 中解析 map[string][]string。
func metadataStringMapSlice(metadata map[string]any, key string) map[string][]string {
	if len(metadata) == 0 {
		return nil
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	normalized := make(map[string][]string)
	switch typed := raw.(type) {
	case map[string][]string:
		for path, values := range typed {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			for _, value := range values {
				value = strings.TrimSpace(value)
				if value == "" {
					continue
				}
				normalized[path] = append(normalized[path], value)
			}
		}
	case map[string]any:
		for path, value := range typed {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			values := metadataStringSlice(map[string]any{"value": value}, "value")
			if len(values) == 0 {
				continue
			}
			normalized[path] = values
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
