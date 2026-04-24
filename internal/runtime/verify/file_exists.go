package verify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

const (
	fileExistsVerifierName = "file_exists"
)

// FileExistsVerifier 校验预期文件是否存在且可用。
type FileExistsVerifier struct{}

// Name 返回 verifier 名称。
func (FileExistsVerifier) Name() string {
	return fileExistsVerifierName
}

// VerifyFinal 校验 metadata 声明的 expected_files 是否存在并且非空文件。
func (FileExistsVerifier) VerifyFinal(_ context.Context, input FinalVerifyInput) (VerificationResult, error) {
	cfg, exists := input.VerificationConfig.Verifiers[fileExistsVerifierName]
	required := exists && cfg.Required
	paths := metadataStringSlice(input.Metadata, "expected_files")
	if len(paths) == 0 {
		if required {
			return VerificationResult{
				Name:    fileExistsVerifierName,
				Status:  VerificationSoftBlock,
				Summary: "expected_files is required but missing",
				Reason:  "missing expected_files metadata",
			}, nil
		}
		return VerificationResult{
			Name:    fileExistsVerifierName,
			Status:  VerificationPass,
			Summary: "no expected files configured, skip file existence check",
			Reason:  "optional verifier skipped",
		}, nil
	}

	missing := make([]string, 0)
	empty := make([]string, 0)
	dirs := make([]string, 0)
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(strings.TrimSpace(input.Workdir), path)
		}
		info, err := os.Stat(absPath)
		if err != nil {
			missing = append(missing, path)
			continue
		}
		if info.IsDir() {
			dirs = append(dirs, path)
			continue
		}
		if info.Size() <= 0 {
			empty = append(empty, path)
		}
	}

	evidence := map[string]any{
		"expected_files":  paths,
		"missing_files":   missing,
		"empty_files":     empty,
		"directory_paths": dirs,
	}
	if len(missing) == 0 && len(empty) == 0 && len(dirs) == 0 {
		return VerificationResult{
			Name:     fileExistsVerifierName,
			Status:   VerificationPass,
			Summary:  "all expected files exist and are non-empty",
			Reason:   "file existence check passed",
			Evidence: evidence,
		}, nil
	}
	return VerificationResult{
		Name:       fileExistsVerifierName,
		Status:     VerificationFail,
		Summary:    "expected files are missing or invalid",
		Reason:     "file existence check failed",
		ErrorClass: ErrorClassUnknown,
		Evidence:   evidence,
	}, nil
}
