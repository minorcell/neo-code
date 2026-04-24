# NeoCode

基于 Go + Bubble Tea 的本地 AI Coding Agent，主链路为：

`用户输入(TUI) -> Gateway -> Runtime -> Tools -> 结果回传 -> UI 展示`

## 产物形态

本项目提供双产物发布：

1. `neocode`：默认完整客户端入口（含 `gateway` 子命令）。
2. `neocode-gateway`：Gateway-Only 服务端入口（不含 TUI 主入口语义）。

## 快速开始

### 能力概览

- 终端原生 TUI 交互体验（Bubble Tea）
- Agent 可调用内置工具完成文件与命令相关任务
- 支持 Provider/Model 切换（内建 `openai`、`gemini`、`openll`、`qiniu`、`modelscope`）
- 支持上下文压缩（`/compact`），帮助长会话保持可用
- 支持工作区隔离（`--workdir`、`/cwd`）
- 会话持久化与恢复，降低重复沟通成本
- 支持持久记忆查看、显式写入与后台自动提取，保留跨会话偏好与项目事实

### 1) 环境要求

- Go `1.25+`
- 可用的 API Key（如 OpenAI、Gemini、OpenLL、Qiniu、ModelScope）

### 2) 从源码运行

```bash
git clone https://github.com/1024XEngineer/neo-code.git
cd neo-code
go run ./cmd/neocode
```

### 3) 启动网关（两种等价方式）

```bash
go run ./cmd/neocode gateway --http-listen 127.0.0.1:8080
```

```bash
go run ./cmd/neocode-gateway --http-listen 127.0.0.1:8080
```

### 4) URL 唤醒分发

```bash
go run ./cmd/neocode url-dispatch --url "neocode://review?path=README.md"
```

当网关不可达时，`url-dispatch` 会按固定发现顺序尝试自动拉起：

1. `NEOCODE_GATEWAY_BIN` 显式路径
2. `PATH` 中 `neocode-gateway`
3. `PATH` 中 `neocode` 并追加子命令 `gateway`

## 安装脚本

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/1024XEngineer/neo-code/main/scripts/install.sh | bash
```

可选 flavor：

```bash
bash ./scripts/install.sh --flavor full
bash ./scripts/install.sh --flavor gateway
```

Dry-run（仅输出资产 URL / checksum URL，不执行下载与安装）：

```bash
bash ./scripts/install.sh --flavor gateway --dry-run
```

Provider API Key 示例（按使用 provider 选择）：

```bash
export OPENAI_API_KEY="your_key_here"
export GEMINI_API_KEY="your_key_here"
export AI_API_KEY="your_key_here"
export QINIU_API_KEY="your_key_here"
export MODELSCOPE_API_KEY="your_key_here"
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/1024XEngineer/neo-code/main/scripts/install.ps1 | iex
```

可选 flavor 与 dry-run：

```powershell
.\scripts\install.ps1 -Flavor full
.\scripts\install.ps1 -Flavor gateway
.\scripts\install.ps1 -Flavor gateway -DryRun
```

Provider API Key 示例（按使用 provider 选择）：

```powershell
$env:OPENAI_API_KEY = "your_key_here"
$env:GEMINI_API_KEY = "your_key_here"
$env:AI_API_KEY = "your_key_here"
$env:QINIU_API_KEY = "your_key_here"
$env:MODELSCOPE_API_KEY = "your_key_here"
```

Gateway 转发与自动拉起说明：

- `neocode` 默认通过本地 Gateway（优先 IPC）转发 runtime 请求与事件流
- 启动时会先探测本地网关；若未运行会自动后台拉起并等待就绪（无感）
- 若自动拉起后仍不可达或握手失败，会直接报错退出（Fail Fast）

### 常用命令

- `/help`：查看命令帮助
- `/provider`：打开 provider 选择器
- `/model`：打开 model 选择器
- `/compact`：压缩当前会话上下文
- `/status`：查看当前会话与运行状态
- `/cwd [path]`：查看或设置当前会话工作区
- `/memo`：查看记忆索引
- `/remember <text>`：保存记忆
- `/forget <keyword>`：按关键词删除记忆
- `/skills`：查看当前可用 skills（含当前会话激活标记）
- `/skill use <id>`：在当前会话启用 skill
- `/skill off <id>`：在当前会话停用 skill
- `/skill active`：查看当前会话已激活 skills

示例输入：

```text
请先阅读当前项目目录结构并给出模块职责摘要
帮我在 internal/runtime 下定位与 tool result 回灌相关逻辑
```

## 部署拓扑建议

1. 本地内嵌（默认）：`neocode` 进程内通过 `gateway` 子命令管理网关。
2. 独立网关服务：使用 `neocode-gateway` 作为可审计、可独立运维的网关进程。

默认监听保持回环地址（`127.0.0.1`）；对外暴露必须显式配置并补齐鉴权与 ACL。

## 升级与回滚（最小流程）

1. 升级后先验证 `GET /healthz`。
2. 再验证 `/rpc` 最小请求（含未鉴权失败路径）。
3. 如异常，回滚到上一个已验证版本的二进制与配置。

## 内部结构补充

- `internal/context`：负责消费仓库/运行时事实并组装主会话 system prompt、动态上下文注入与消息裁剪。
- `internal/repository`：负责仓库级事实发现与裁剪，统一提供 repo summary、changed-files context 与 targeted retrieval。
- `internal/runtime`：负责 ReAct 主循环、tool 调用编排、compact 触发与 reminder 注入时机。
- `internal/subagent`：负责子代理角色策略、执行约束与输出契约。
- `internal/promptasset`：负责受版本管理的静态 prompt 模板资产，使用 `go:embed` 编译进程序，供 `context`、`runtime`、`subagent` 读取。

## 文档索引

- [Gateway 详细设计 RFC](docs/gateway-detailed-design.md)
- [Gateway 第三方接入协作指南](docs/guides/gateway-integration-guide.md)
- [Gateway RPC API（XGO 风格）](docs/gateway-rpc-api.md)
- [Gateway 错误字典](docs/gateway-error-catalog.md)
- [Gateway 兼容性策略](docs/gateway-compatibility.md)
- [配置指南](docs/guides/configuration.md)
- [扩展 Provider](docs/guides/adding-providers.md)
- [Runtime/Provider 事件流](docs/runtime-provider-event-flow.md)
- [Session 持久化设计](docs/session-persistence-design.md)
- [Context Compact 说明](docs/context-compact.md)
- [Repository 模块设计](docs/repository-design.md)
- [Tools 与 TUI 集成](docs/tools-and-tui-integration.md)
- [Skills 设计与使用](docs/skills-system-design.md)
- [MCP 配置指南](docs/guides/mcp-configuration.md)
- [ModelScope 半引导配置](docs/guides/modelscope-provider-setup.md)
- [更新指南（更新与升级）](docs/guides/update.md)

## 如何参与

欢迎通过 Issue 和 PR 参与共建。

1. 在 [Issues](https://github.com/1024XEngineer/neo-code/issues) 先沟通问题或需求。
2. Fork 仓库并创建功能分支。
3. 完成开发并确保改动聚焦、边界清晰。
4. 本地自检：
   ```bash
   make docs-gateway-check
   gofmt -w ./cmd ./internal
   go test ./...
   go build ./...
   ```
5. 提交 PR 到主仓库并说明变更目的、影响范围和验证方式。

提交前请确认：

- 不提交明文密钥、个人配置或会话数据
- 不提交无关改动与临时文件

## 在仓库内直接创建 Issue（Skills + 自动化）

仓库提供三类同前缀 skill（位于 `.agents/skills/`）：

- `issue-rfc-proposal`（提案类，RFC 风格）
- `issue-rfc-architecture`（架构类，RFC 风格）
- `issue-rfc-implementation`（实现类，执行单风格）

## 开发与验证

```bash
go build ./...
go test ./...
gofmt -w ./cmd ./internal
```

## License

MIT
