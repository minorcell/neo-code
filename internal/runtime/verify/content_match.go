package verify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

const (
	contentMatchVerifierName = "content_match"
)

// ContentMatchVerifier 校验预期文件内容是否命中要求。
type ContentMatchVerifier struct{}

// Name 返回 verifier 名称。
func (ContentMatchVerifier) Name() string {
	return contentMatchVerifierName
}

// VerifyFinal 校验 metadata.content_match 声明的路径与关键内容约束。
func (ContentMatchVerifier) VerifyFinal(_ context.Context, input FinalVerifyInput) (VerificationResult, error) {
	cfg, exists := input.VerificationConfig.Verifiers[contentMatchVerifierName]
	required := exists && cfg.Required
	rules := metadataStringMapSlice(input.Metadata, "content_match")
	if len(rules) == 0 {
		if required {
			return VerificationResult{
				Name:    contentMatchVerifierName,
				Status:  VerificationSoftBlock,
				Summary: "content_match is required but missing",
				Reason:  "missing content_match metadata",
			}, nil
		}
		return VerificationResult{
			Name:    contentMatchVerifierName,
			Status:  VerificationPass,
			Summary: "no content_match rule configured, skip content check",
			Reason:  "optional verifier skipped",
		}, nil
	}

	missingFiles := make([]string, 0)
	missingTokens := make(map[string][]string)
	for rawPath, expectedTokens := range rules {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(strings.TrimSpace(input.Workdir), path)
		}
		contentBytes, err := os.ReadFile(absPath)
		if err != nil {
			missingFiles = append(missingFiles, path)
			continue
		}
		content := string(contentBytes)
		for _, token := range expectedTokens {
			normalized := strings.TrimSpace(token)
			if normalized == "" {
				continue
			}
			if !strings.Contains(content, normalized) {
				missingTokens[path] = append(missingTokens[path], normalized)
			}
		}
	}

	evidence := map[string]any{
		"rules":          rules,
		"missing_files":  missingFiles,
		"missing_tokens": missingTokens,
	}
	if len(missingFiles) == 0 && len(missingTokens) == 0 {
		return VerificationResult{
			Name:     contentMatchVerifierName,
			Status:   VerificationPass,
			Summary:  "all expected content rules matched",
			Reason:   "content match check passed",
			Evidence: evidence,
		}, nil
	}
	return VerificationResult{
		Name:       contentMatchVerifierName,
		Status:     VerificationFail,
		Summary:    "content rule mismatch detected",
		Reason:     "content match check failed",
		ErrorClass: ErrorClassUnknown,
		Evidence:   evidence,
	}, nil
}
