# 更新与版本检查

## 自动检查
- `neocode` 启动时会在后台静默检查最新稳定版本（默认 3 秒超时）。
- 为避免干扰 Bubble Tea TUI 交互，更新提示会在应用退出、终端屏幕恢复后输出。
- `url-dispatch`、`update`、`version` 子命令会跳过该静默检查，避免重复探测。

## 查询版本

查看当前版本并探测远端最新版本：

```bash
neocode version
```

包含预发布版本一起比较：

```bash
neocode version --prerelease
```

行为说明：
- 始终输出当前版本。
- 探测成功时输出“最新版本 + 比较结果”。
- 探测失败时输出失败原因，但命令仍返回成功退出码，方便脚本场景继续执行。

## 手动升级

升级到最新稳定版本：

```bash
neocode update
```

包含预发布版本：

```bash
neocode update --prerelease
```

更新命令在平台资产匹配失败时会输出可诊断信息，例如：
- `os`
- `arch`
- `expected-pattern`
- `available-assets-count`
- `candidate-assets`（最多展示前 10 个，单项最长 120 字符）

## 版本来源

- 发布构建会通过 `ldflags` 注入版本号到 `internal/version.Version`。
- 本地开发构建默认版本为 `dev`。
