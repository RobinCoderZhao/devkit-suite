# 开发计划 — Phase 2：开发者 CLI 工具套件（DevKit）

> 前置依赖：Phase 0（`pkg/llm` 必须完成）
>
> 项目路径：`devkit-suite/cmd/devkit/` + `devkit-suite/internal/devkit/`

---

## Step 2.1：目录结构

```
cmd/devkit/
└── main.go                         # Cobra 根命令初始化

internal/devkit/
├── cmd/                            # 各子命令实现
│   ├── root.go                     # 根命令 + 全局 flags
│   ├── commit.go                   # devkit commit
│   ├── review.go                   # devkit review
│   ├── doc.go                      # devkit doc (V1.1)
│   ├── test.go                     # devkit test (V1.1)
│   ├── changelog.go                # devkit changelog (V1.2)
│   └── translate.go                # devkit translate (V1.2)
├── git/                            # Git 操作封装
│   ├── git.go                      # GitOps 接口 + 实现
│   └── git_test.go
├── prompt/                         # 各命令的 Prompt 模板
│   ├── commit_prompt.go
│   ├── review_prompt.go
│   ├── doc_prompt.go
│   └── test_prompt.go
├── ui/                             # 终端交互 UI
│   ├── spinner.go                  # 加载动画
│   ├── select.go                   # 交互式选择
│   ├── editor.go                   # 调用 $EDITOR
│   ├── color.go                    # 彩色输出
│   └── ui_test.go
├── config/                         # DevKit 专用配置
│   ├── config.go                   # ~/.devkit.yaml 加载
│   └── defaults.go                 # 默认值
└── license/                        # License 验证
    ├── license.go                  # 验证逻辑
    └── license_test.go
```

## Step 2.2：Cobra 命令树

```go
// cmd/devkit/main.go
package main

func main() {
    cmd.Execute()
}

// internal/devkit/cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "devkit",
    Short: "AI-powered developer CLI toolkit",
    Long:  "DevKit provides AI-assisted tools for everyday development tasks.",
}

func init() {
    rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default ~/.devkit.yaml)")
    rootCmd.PersistentFlags().StringP("provider", "p", "", "LLM provider override")
    rootCmd.PersistentFlags().StringP("model", "m", "", "model override")
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

    rootCmd.AddCommand(commitCmd)
    rootCmd.AddCommand(reviewCmd)
    // V1.1: rootCmd.AddCommand(docCmd, testCmd)
    // V1.2: rootCmd.AddCommand(changelogCmd, translateCmd)
}

func Execute() { rootCmd.Execute() }
```

## Step 2.3：`devkit commit` 完整实现规格

```go
// internal/devkit/cmd/commit.go
var commitCmd = &cobra.Command{
    Use:   "commit",
    Short: "Generate AI-powered commit messages",
    Long:  "Analyzes staged changes and generates conventional commit messages using AI.",
    RunE:  runCommit,
}

func init() {
    commitCmd.Flags().StringP("language", "l", "en", "message language (en/zh/ja)")
    commitCmd.Flags().StringP("format", "f", "conventional", "format (conventional/simple)")
    commitCmd.Flags().IntP("max-length", "n", 72, "max title length")
    commitCmd.Flags().BoolP("auto", "a", false, "auto-commit without confirmation")
    commitCmd.Flags().BoolP("body", "b", true, "include commit body")
}

// 完整流程:
// 1. git.IsRepo() → 检查是否在 git 仓库
// 2. git.GetStagedDiff() → 获取 staged changes
// 3. len(diff) 检测 → 如果为空，提示 "no staged changes"
// 4. truncateDiff(diff, 8000) → 大 diff 智能截断
// 5. git.GetRecentCommits(5) → 获取近 5 条 commit（学习风格）
// 6. buildPrompt() → 组装 prompt
// 7. llm.Generate() → 调用 LLM（带 spinner）
// 8. 展示结果 → Accept / Edit / Regenerate / Cancel
// 9. git.DoCommit(msg) → 执行 commit
```

### Commit Prompt 模板

```go
// internal/devkit/prompt/commit_prompt.go
const CommitPrompt = `Based on the following git diff, generate a commit message.

Format: {{.Format}} commits (e.g., "feat: add user login", "fix: resolve null pointer")
Language: {{.Language}}
Max title length: {{.MaxLength}} characters

Recent commits for style reference:
{{range .RecentCommits}}
- {{.Message}}
{{end}}

Git diff:
` + "```" + `
{{.Diff}}
` + "```" + `

Rules:
1. Title must be one line, starting with type prefix (feat/fix/refactor/docs/test/chore)
2. Title should be imperative mood ("add" not "added")
3. Body should explain WHY, not WHAT (the diff shows WHAT)
4. If multiple changes, summarize the main intent

Output JSON:
{
  "title": "feat: ...",
  "body": "Optional detailed explanation...",
  "type": "feat|fix|refactor|docs|test|chore"
}`
```

## Step 2.4：`devkit review` 完整实现规格

```go
// internal/devkit/cmd/review.go
var reviewCmd = &cobra.Command{
    Use:   "review [file...]",
    Short: "AI-powered code review",
    Long:  "Reviews staged changes or specified files for potential issues.",
    RunE:  runReview,
}

func init() {
    reviewCmd.Flags().StringP("focus", "", "all", "focus areas: security,performance,error-handling,all")
    reviewCmd.Flags().StringP("output", "o", "terminal", "output: terminal/markdown/json")
    reviewCmd.Flags().BoolP("staged", "s", true, "review staged changes")
    reviewCmd.Flags().StringP("branch", "", "", "review changes against branch")
}

// 流程：
// 1. 获取 diff（staged / branch / file list）
// 2. 按文件分割 diff
// 3. 每个文件分别调用 LLM review
// 4. 合并结果 + 打分
// 5. 按 severity 排序输出

// 输出格式：
// ╔══════════════════════════════════════╗
// ║  Code Review Report    Score: 7/10  ║
// ╠══════════════════════════════════════╣
// ║                                      ║
// ║  🔴 ERROR  api/handler.go:45         ║
// ║  Missing error handling for query    ║
// ║  Suggested fix: ...                  ║
// ║                                      ║
// ║  🟡 WARN   utils/parse.go:12        ║
// ║  Potential nil pointer dereference   ║
// ║                                      ║
// ╚══════════════════════════════════════╝
```

## Step 2.5：`devkit doc` 实现规格 (V1.1)

```go
// 输入：指定的 Go 文件 或 目录
// 输出：Markdown 格式的 API 文档

// 流程：
// 1. 解析 Go 源码（go/parser + go/ast）
// 2. 提取: package / exported functions / types / methods / comments
// 3. 将 AST 信息传给 LLM 补充描述
// 4. 生成 Markdown 文档输出到 stdout 或文件
```

## Step 2.6：`devkit test` 实现规格 (V1.1)

```go
// 输入：指定的 Go 文件
// 输出：对应的 _test.go 文件内容

// 流程：
// 1. 读取源文件内容
// 2. 提取所有 exported functions
// 3. LLM 生成 table-driven 测试
// 4. 输出或写入 xxx_test.go
```

## Step 2.7：Git 操作封装

```go
// internal/devkit/git/git.go
type GitOps struct {
    repoPath string
}

func New(path string) *GitOps
func (g *GitOps) IsRepo() bool
func (g *GitOps) GetStagedDiff() (string, error)
func (g *GitOps) GetRecentCommits(n int) ([]Commit, error)
func (g *GitOps) GetChangedFiles() ([]string, error)
func (g *GitOps) DoCommit(message string) error
func (g *GitOps) GetFileContent(path, ref string) (string, error)
func (g *GitOps) GetDiffBetween(from, to string) (string, error)
func (g *GitOps) GetCurrentBranch() (string, error)
func (g *GitOps) GetRemoteURL() (string, error)

type Commit struct {
    Hash    string
    Message string
    Author  string
    Date    time.Time
}

// 底层实现：全部使用 os/exec 调用 git 命令
// 不引入 go-git 库，减少依赖
```

## Step 2.8：终端 UI 组件

```go
// internal/devkit/ui/
// 使用的库：github.com/charmbracelet/bubbletea + lipgloss

// spinner.go   — 等待 LLM 响应时的加载动画
// select.go    — 多选一（Accept/Edit/Regenerate/Cancel）
// color.go     — 彩色输出：Green(✅) Yellow(🟡) Red(🔴)
// editor.go    — 调用 $EDITOR 或 $VISUAL 编辑文本
```

### 依赖

```
go get github.com/spf13/cobra
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
```

## Step 2.9：配置文件

```yaml
# ~/.devkit.yaml
llm:
  provider: openai
  model: gpt-4o-mini
  api_key: ""                    # 或 DEVKIT_API_KEY 环境变量

commit:
  language: en
  format: conventional
  max_length: 72
  include_body: true

review:
  focus:
    - security
    - performance
    - error-handling
  output: terminal

# Pro License
license:
  key: ""                        # DEVKIT_LICENSE_KEY 环境变量
```

## Step 2.10：GoReleaser + 分发

```yaml
# .goreleaser.yml
version: 2
builds:
  - id: devkit
    main: ./cmd/devkit
    binary: devkit
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env: [CGO_ENABLED=0]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}

brews:
  - repository:
      owner: RobinCoderZhao
      name: homebrew-tap
    homepage: "https://github.com/RobinCoderZhao/devkit-suite"
    description: "AI-powered developer CLI toolkit"
    install: bin.install "devkit"

archives:
  - id: devkit
    builds: [devkit]
    format: tar.gz
    name_template: "devkit_{{ .Os }}_{{ .Arch }}"

changelog:
  sort: asc
```

## Step 2.11：开发顺序 & 验证

| 序号 | 任务 | 验证标准 | 预计时间 |
|------|------|---------|---------|
| 1 | git 操作封装 | `GetStagedDiff` / `DoCommit` 测试通过 | 2h |
| 2 | UI 组件实现 | spinner / select 可交互 | 2h |
| 3 | 配置加载 | `~/.devkit.yaml` 读取正确 | 1h |
| 4 | `devkit commit` 完整实现 | 从 staged diff → 生成 → 确认 → 提交 | 3h |
| 5 | `devkit review` 完整实现 | 输出结构化 review 报告 | 3h |
| 6 | License 验证 | Free/Pro 区分正确 | 1h |
| 7 | GoReleaser 打包 | 生成 macOS/Linux/Windows 二进制 | 1h |
| 8 | README + demo GIF | asciinema 录制 | 1h |
| **总计** | | | **约 14h（2-3 天）** |

## 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| Git 操作方式 | `os/exec` 调用 git | 零依赖、用户机器都有 git |
| CLI 框架 | Cobra | Go 生态标准 |
| UI 库 | Bubbletea + Lipgloss | 美观、现代、社区大 |
| 配置格式 | YAML | 人类友好 |
| Diff 截断策略 | 保留文件头 + 首尾变更 | LLM context 有限 |
