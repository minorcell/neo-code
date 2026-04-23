package updater

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	selfupdate "github.com/creativeprojects/go-selfupdate"

	"neo-code/internal/version"
)

const (
	repositoryOwner  = "1024XEngineer"
	repositoryName   = "neo-code"
	checksumFilename = "checksums.txt"

	maxDiagnosticCandidateAssets = 10
	maxDiagnosticAssetNameLength = 120
)

var (
	runtimeGOOS   = runtime.GOOS
	runtimeGOARCH = runtime.GOARCH
)

var (
	newClient = func(config selfupdate.Config) (updateClient, error) {
		source := config.Source
		if source == nil {
			created, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
			if err != nil {
				return nil, err
			}
			source = created
		}
		config.Source = source

		updater, err := selfupdate.NewUpdater(config)
		if err != nil {
			return nil, err
		}
		return selfupdateClient{
			updater: updater,
			source:  source,
			config:  config,
		}, nil
	}
	resolveExecutablePath = selfupdate.ExecutablePath
)

var semverTagPattern = regexp.MustCompile(`\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?`)

type assetTarget struct {
	OSToken   string
	ArchToken string
	Ext       string
	AssetName string
}

type probeStatus uint8

const (
	probeStatusMatched probeStatus = iota + 1
	probeStatusNoCandidate
	probeStatusAmbiguous
)

type probeResult struct {
	Status               probeStatus
	Release              releaseView
	LatestVersion        string
	ExpectedPattern      string
	AvailableAssetsCount int
	CandidateAssets      []string
}

type releaseView interface {
	Version() string
	GreaterThan(other string) bool
}

type updateClient interface {
	ProbeLatest(ctx context.Context, repository selfupdate.Repository, target assetTarget) (probeResult, error)
	UpdateTo(ctx context.Context, rel releaseView, cmdPath string) error
}

type selfupdateClient struct {
	updater *selfupdate.Updater
	source  selfupdate.Source
	config  selfupdate.Config
}

type selfupdateRelease struct {
	release *selfupdate.Release
}

type releaseSnapshot struct {
	Release       selfupdate.SourceRelease
	Version       *semver.Version
	MatchedAssets []selfupdate.SourceAsset
}

// CheckOptions 描述静默探测最新版本时的输入参数。
type CheckOptions struct {
	CurrentVersion    string
	IncludePrerelease bool
}

// CheckResult 表示静默探测流程返回的版本信息。
type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
}

// UpdateOptions 描述手动更新命令的输入参数。
type UpdateOptions struct {
	CurrentVersion    string
	IncludePrerelease bool
}

// UpdateResult 表示手动更新流程的最终结果。
type UpdateResult struct {
	CurrentVersion string
	LatestVersion  string
	Updated        bool
}

// CheckLatest 按当前平台语义规则探测最新版本，不做本地文件替换。
func CheckLatest(ctx context.Context, opts CheckOptions) (CheckResult, error) {
	currentVersion := normalizeCurrentVersion(opts.CurrentVersion)
	target, err := resolveAssetTarget(runtimeGOOS, runtimeGOARCH)
	if err != nil {
		return CheckResult{CurrentVersion: currentVersion}, err
	}

	client, err := newClient(buildSelfupdateConfig(target, opts.IncludePrerelease))
	if err != nil {
		return CheckResult{CurrentVersion: currentVersion}, err
	}

	repository := selfupdate.NewRepositorySlug(repositoryOwner, repositoryName)
	probe, err := client.ProbeLatest(ctx, repository, target)
	if err != nil {
		return CheckResult{CurrentVersion: currentVersion}, err
	}

	result := CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  strings.TrimSpace(probe.LatestVersion),
	}
	if result.LatestVersion == "" {
		return result, nil
	}
	if probe.Status != probeStatusMatched || probe.Release == nil {
		return result, nil
	}

	if version.IsSemverRelease(currentVersion) {
		result.HasUpdate = probe.Release.GreaterThan(currentVersion)
	}
	return result, nil
}

// DoUpdate 下载并校验最新版本后原地替换当前可执行文件。
func DoUpdate(ctx context.Context, opts UpdateOptions) (UpdateResult, error) {
	currentVersion := normalizeCurrentVersion(opts.CurrentVersion)
	target, err := resolveAssetTarget(runtimeGOOS, runtimeGOARCH)
	if err != nil {
		return UpdateResult{CurrentVersion: currentVersion}, err
	}

	client, err := newClient(buildSelfupdateConfig(target, opts.IncludePrerelease))
	if err != nil {
		return UpdateResult{CurrentVersion: currentVersion}, err
	}

	repository := selfupdate.NewRepositorySlug(repositoryOwner, repositoryName)
	probe, err := client.ProbeLatest(ctx, repository, target)
	if err != nil {
		return UpdateResult{CurrentVersion: currentVersion}, err
	}

	result := UpdateResult{
		CurrentVersion: currentVersion,
		LatestVersion:  strings.TrimSpace(probe.LatestVersion),
	}

	switch probe.Status {
	case probeStatusNoCandidate:
		return result, newAssetDiagnosticError("updater: no release asset found for current platform", target, probe)
	case probeStatusAmbiguous:
		return result, newAssetDiagnosticError("updater: multiple release assets matched current platform", target, probe)
	case probeStatusMatched:
		if probe.Release == nil {
			return result, newAssetDiagnosticError("updater: no release asset found for current platform", target, probe)
		}
	default:
		return result, newAssetDiagnosticError("updater: no release asset found for current platform", target, probe)
	}

	if version.IsSemverRelease(currentVersion) && !probe.Release.GreaterThan(currentVersion) {
		return result, nil
	}

	executablePath, err := resolveExecutablePath()
	if err != nil {
		return result, err
	}

	if err := client.UpdateTo(ctx, probe.Release, executablePath); err != nil {
		return result, err
	}

	result.Updated = true
	return result, nil
}

// ProbeLatest 以平台语义匹配策略探测最新可用资产，并输出可诊断元数据。
func (c selfupdateClient) ProbeLatest(
	ctx context.Context,
	repository selfupdate.Repository,
	target assetTarget,
) (probeResult, error) {
	result := probeResult{
		Status:          probeStatusNoCandidate,
		ExpectedPattern: buildExpectedPattern(target),
	}

	releases, err := c.source.ListReleases(ctx, repository)
	if err != nil {
		return result, err
	}

	matcher := regexp.MustCompile(result.ExpectedPattern)
	var latestEligible *releaseSnapshot
	var latestMatched *releaseSnapshot

	for _, rel := range releases {
		snapshot, ok := buildReleaseSnapshot(rel, c.config.Prerelease, matcher)
		if !ok {
			continue
		}
		if latestEligible == nil || snapshot.Version.GreaterThan(latestEligible.Version) {
			latestEligible = snapshot
		}
		if len(snapshot.MatchedAssets) == 0 {
			continue
		}
		if latestMatched == nil || snapshot.Version.GreaterThan(latestMatched.Version) {
			latestMatched = snapshot
		}
	}

	if latestEligible != nil {
		result.LatestVersion = latestEligible.Version.String()
	}
	if latestMatched == nil {
		if latestEligible != nil {
			allAssets := collectAssetNames(latestEligible.Release.GetAssets())
			result.AvailableAssetsCount = len(allAssets)
			result.CandidateAssets = sampleAssetsForDiagnostic(allAssets)
		}
		return result, nil
	}

	result.LatestVersion = latestMatched.Version.String()
	result.AvailableAssetsCount = len(latestMatched.Release.GetAssets())

	matchedNames := collectAssetNames(latestMatched.MatchedAssets)
	result.CandidateAssets = sampleAssetsForDiagnostic(matchedNames)
	if len(latestMatched.MatchedAssets) > 1 {
		result.Status = probeStatusAmbiguous
		return result, nil
	}

	chosenAsset := firstNonEmptyAssetName(latestMatched.MatchedAssets)
	release, found, err := c.detectReleaseByTagAndAsset(
		ctx,
		repository,
		latestMatched.Release.GetTagName(),
		chosenAsset,
		target,
	)
	if err != nil {
		return result, err
	}
	if !found || release == nil {
		return result, nil
	}

	result.Status = probeStatusMatched
	result.Release = release
	result.LatestVersion = strings.TrimSpace(release.Version())
	return result, nil
}

// DetectLatest 调用底层 go-selfupdate 客户端获取最新版本信息。
func (c selfupdateClient) DetectLatest(ctx context.Context, repository selfupdate.Repository) (releaseView, bool, error) {
	release, found, err := c.updater.DetectLatest(ctx, repository)
	if err != nil || !found || release == nil {
		return nil, found, err
	}
	return selfupdateRelease{release: release}, true, nil
}

// UpdateTo 委托 go-selfupdate 完成原地替换流程，不追加平台分支逻辑。
func (c selfupdateClient) UpdateTo(ctx context.Context, rel releaseView, cmdPath string) error {
	typed, ok := rel.(selfupdateRelease)
	if !ok || typed.release == nil {
		return errors.New("updater: unsupported release type")
	}
	return c.updater.UpdateTo(ctx, typed.release, cmdPath)
}

// detectReleaseByTagAndAsset 在已确定 tag 和资产名后，按精确资产过滤拿到可下载 release。
func (c selfupdateClient) detectReleaseByTagAndAsset(
	ctx context.Context,
	repository selfupdate.Repository,
	tagName string,
	assetName string,
	target assetTarget,
) (releaseView, bool, error) {
	cleanTag := strings.TrimSpace(tagName)
	cleanAsset := strings.ToLower(strings.TrimSpace(assetName))
	if cleanTag == "" || cleanAsset == "" {
		return nil, false, nil
	}

	config := selfupdate.Config{
		Source:        c.source,
		Validator:     c.config.Validator,
		Filters:       []string{"^" + regexp.QuoteMeta(cleanAsset) + "$"},
		OS:            target.OSToken,
		Arch:          target.ArchToken,
		Arm:           c.config.Arm,
		UniversalArch: c.config.UniversalArch,
		Draft:         c.config.Draft,
		Prerelease:    c.config.Prerelease,
		OldSavePath:   c.config.OldSavePath,
	}
	updater, err := selfupdate.NewUpdater(config)
	if err != nil {
		return nil, false, err
	}

	release, found, err := updater.DetectVersion(ctx, repository, cleanTag)
	if err != nil || !found || release == nil {
		return nil, found, err
	}
	return selfupdateRelease{release: release}, true, nil
}

// Version 返回底层 release 的语义化版本字符串。
func (r selfupdateRelease) Version() string {
	return strings.TrimSpace(r.release.Version())
}

// GreaterThan 判断底层 release 是否高于指定版本。
func (r selfupdateRelease) GreaterThan(other string) bool {
	return r.release.GreaterThan(other)
}

// normalizeCurrentVersion 归一化当前版本输入并处理空值回退。
func normalizeCurrentVersion(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "dev"
	}
	return trimmed
}

// buildSelfupdateConfig 构建更新客户端配置，默认按平台语义匹配，不绑定精确资产名。
func buildSelfupdateConfig(target assetTarget, includePrerelease bool) selfupdate.Config {
	return selfupdate.Config{
		OS:         target.OSToken,
		Arch:       target.ArchToken,
		Validator:  &selfupdate.ChecksumValidator{UniqueFilename: checksumFilename},
		Prerelease: includePrerelease,
	}
}

// resolveAssetTarget 按平台信息归一化出资产语义匹配目标。
func resolveAssetTarget(goos string, goarch string) (assetTarget, error) {
	var osToken string
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "linux":
		osToken = "linux"
	case "darwin":
		osToken = "darwin"
	case "windows":
		osToken = "windows"
	default:
		return assetTarget{}, fmt.Errorf("updater: unsupported os %q", goos)
	}

	var archToken string
	switch strings.ToLower(strings.TrimSpace(goarch)) {
	case "amd64":
		archToken = "x86_64"
	case "arm64":
		archToken = "arm64"
	default:
		return assetTarget{}, fmt.Errorf("updater: unsupported arch %q", goarch)
	}

	ext := "tar.gz"
	if osToken == "windows" {
		ext = "zip"
	}

	return assetTarget{
		OSToken:   osToken,
		ArchToken: archToken,
		Ext:       ext,
		AssetName: fmt.Sprintf("neocode_%s_%s.%s", osToken, archToken, ext),
	}, nil
}

// buildReleaseSnapshot 将 source release 归一化为可比较快照，并过滤不可用发布。
func buildReleaseSnapshot(
	release selfupdate.SourceRelease,
	includePrerelease bool,
	matcher *regexp.Regexp,
) (*releaseSnapshot, bool) {
	if release == nil || release.GetDraft() {
		return nil, false
	}
	if release.GetPrerelease() && !includePrerelease {
		return nil, false
	}

	parsedVersion, ok := parseReleaseVersion(release.GetTagName())
	if !ok {
		return nil, false
	}

	matched := make([]selfupdate.SourceAsset, 0, len(release.GetAssets()))
	for _, asset := range release.GetAssets() {
		name := strings.ToLower(strings.TrimSpace(assetName(asset)))
		if name == "" || !matcher.MatchString(name) {
			continue
		}
		matched = append(matched, asset)
	}

	return &releaseSnapshot{
		Release:       release,
		Version:       parsedVersion,
		MatchedAssets: matched,
	}, true
}

// parseReleaseVersion 从 tag 中提取可比较语义化版本，兼容前缀字符。
func parseReleaseVersion(tag string) (*semver.Version, bool) {
	trimmed := strings.TrimSpace(tag)
	if trimmed == "" {
		return nil, false
	}
	raw := semverTagPattern.FindString(trimmed)
	if raw == "" {
		return nil, false
	}
	parsed, err := semver.NewVersion(raw)
	if err != nil {
		return nil, false
	}
	return parsed, true
}

// buildExpectedPattern 构建平台语义匹配模式，允许大小写和分隔符变体。
func buildExpectedPattern(target assetTarget) string {
	osPattern := platformAliasPattern(target.OSToken)
	archPattern := archAliasPattern(target.ArchToken)
	extPattern := extAliasPattern(target.Ext)
	return fmt.Sprintf(
		`^neocode[-_]%s[-_]%s(?:[-_.][0-9a-z]+)*(?:\.exe)?(?:%s)$`,
		osPattern,
		archPattern,
		extPattern,
	)
}

// platformAliasPattern 返回平台别名匹配表达式。
func platformAliasPattern(osToken string) string {
	switch strings.ToLower(strings.TrimSpace(osToken)) {
	case "windows":
		return `(?:windows|win)`
	case "darwin":
		return `(?:darwin|macos)`
	default:
		return regexp.QuoteMeta(strings.ToLower(strings.TrimSpace(osToken)))
	}
}

// archAliasPattern 返回架构别名匹配表达式。
func archAliasPattern(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x86_64":
		return `(?:x86_64|x86-64|amd64)`
	case "arm64":
		return `(?:arm64|aarch64)`
	default:
		return regexp.QuoteMeta(strings.ToLower(strings.TrimSpace(arch)))
	}
}

// extAliasPattern 返回归档扩展名匹配表达式。
func extAliasPattern(ext string) string {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case "tar.gz":
		return `(?:\.tar\.gz|\.tgz)`
	default:
		cleaned := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), ".")
		return `(?:\.` + regexp.QuoteMeta(cleaned) + `)`
	}
}

// newAssetDiagnosticError 生成包含平台与候选信息的可执行诊断错误。
func newAssetDiagnosticError(message string, target assetTarget, probe probeResult) error {
	return fmt.Errorf(
		`%s (os=%s arch=%s expected-pattern="%s" available-assets-count=%d candidate-assets=%v)`,
		message,
		target.OSToken,
		target.ArchToken,
		probe.ExpectedPattern,
		probe.AvailableAssetsCount,
		probe.CandidateAssets,
	)
}

// collectAssetNames 提取并排序资产名，便于稳定输出诊断信息。
func collectAssetNames(assets []selfupdate.SourceAsset) []string {
	names := make([]string, 0, len(assets))
	for _, asset := range assets {
		name := strings.TrimSpace(assetName(asset))
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sampleAssetsForDiagnostic 按上限截断候选资产，防止诊断输出失控。
func sampleAssetsForDiagnostic(names []string) []string {
	sampled := make([]string, 0, minInt(len(names), maxDiagnosticCandidateAssets))
	for _, name := range names {
		if len(sampled) >= maxDiagnosticCandidateAssets {
			break
		}
		sampled = append(sampled, trimDiagnosticAssetName(name))
	}
	return sampled
}

// trimDiagnosticAssetName 对候选资产名按长度截断，控制日志噪声。
func trimDiagnosticAssetName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxDiagnosticAssetNameLength {
		return trimmed
	}
	return string(runes[:maxDiagnosticAssetNameLength]) + "..."
}

// firstNonEmptyAssetName 返回第一个可用资产名，用于二次精确探测。
func firstNonEmptyAssetName(assets []selfupdate.SourceAsset) string {
	for _, asset := range assets {
		name := strings.TrimSpace(assetName(asset))
		if name != "" {
			return name
		}
	}
	return ""
}

// assetName 统一提取资产展示名，优先使用资产名，缺失时回退下载 URL。
func assetName(asset selfupdate.SourceAsset) string {
	if asset == nil {
		return ""
	}
	name := strings.TrimSpace(asset.GetName())
	if name != "" {
		return name
	}
	return strings.TrimSpace(asset.GetBrowserDownloadURL())
}

// minInt 返回两个整数中的较小值。
func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
