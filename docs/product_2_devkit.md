# 产品 2：开发者 CLI 工具套件（DevKit）详细设计

## 1. 产品定义

### 1.1 产品愿景
>
> "一个 CLI，搞定所有 AI 辅助开发任务。Go 写的，快如闪电。"

### 1.2 目标用户

| 画像 | 描述 | 付费意愿 |
|------|------|---------|
| **后端工程师** | 日常写代码、提交 PR | 中（省时间 = 愿意付钱） |
| **全栈开发者** | 需要快速生成文档和测试 | 高 |
| **DevOps/SRE** | 希望自动化 commit/changelog | 中 |
| **开源维护者** | 需要翻译 README、生成 changelog | 高 |

### 1.3 命令矩阵

| 命令 | 功能 | 版本 | 竞品差距 |
|------|------|------|---------|
| `devkit commit` | AI 生成 commit message | MVP | aicommits 仅支持 Node.js |
| `devkit review` | AI Code Review | MVP | 无同类 CLI 工具 |
| `devkit doc` | 从代码生成 API 文档 | V1.1 | 手动写文档太慢 |
| `devkit test` | AI 生成单元测试 | V1.1 | 目前只有 IDE 插件 |
| `devkit changelog` | 从 git log 生成 CHANGELOG | V1.2 | 现有工具不用 AI |
| `devkit translate` | 翻译 README/文档 | V1.2 | 没有保留格式的工具 |

---

## 2. 软件架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────┐
│                   CLI Layer (cobra)              │
│   devkit commit │ review │ doc │ test │ ...      │
└────────────┬────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────┐
│              Command Handlers                     │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│  │ commit   │ │ review  │ │  doc    │  ...      │
│  │ handler  │ │ handler │ │ handler │           │
│  └────┬─────┘ └────┬────┘ └────┬────┘           │
└───────┼─────────────┼──────────┼────────────────┘
        │             │          │
┌───────▼─────────────▼──────────▼────────────────┐
│              Core Services                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ Git Ops  │ │ LLM      │ │ Config Manager   │ │
│  │ (diff,   │ │ Client   │ │ (~/.devkit.yaml) │ │
│  │  log,    │ │ (多模型)  │ │                  │ │
│  │  stage)  │ │          │ │                  │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
└───────────────────────┬─────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────┐
│            License Layer (可选)                   │
│   Free: commit + review(5次/天)                  │
│   Pro:  全部命令 + 无限次数 + 团队配置             │
└─────────────────────────────────────────────────┘
```

### 2.2 核心模块

#### Git Operations

```go
// internal/devkit/git/git.go
type GitOps struct {
    repoPath string
}

func (g *GitOps) GetStagedDiff() (string, error)            // git diff --staged
func (g *GitOps) GetRecentCommits(n int) ([]Commit, error)   // git log --oneline -n
func (g *GitOps) GetChangedFiles() ([]string, error)         // git status --porcelain
func (g *GitOps) DoCommit(message string) error              // git commit -m
func (g *GitOps) GetFileContent(path, ref string) (string, error) // git show ref:path
func (g *GitOps) GetDiffBetween(from, to string) (string, error)  // git diff from..to
```

#### Config Manager

```yaml
# ~/.devkit.yaml（用户全局配置）
llm:
  provider: openai              # openai / gemini / claude / ollama
  model: gpt-4o-mini            # 默认模型
  api_key: sk-xxx               # 或从环境变量读取
  base_url: ""                  # 自定义 endpoint（ollama 必填）

commit:
  language: en                  # 生成语言：en / zh / ja / auto
  format: conventional          # conventional / simple / angular
  max_length: 72                # 标题最大长度
  include_body: true            # 是否生成 body

review:
  focus:                        # 重点关注
    - security
    - performance
    - error-handling
  output: markdown              # markdown / plain / json

license:
  key: ""                       # Pro License Key
```

#### Commit 命令完整流程

```go
// internal/devkit/cmd/commit.go
func runCommit(cmd *cobra.Command, args []string) error {
    // 1. 检查是否在 git 仓库中
    git := gitops.New(".")
    if !git.IsRepo() {
        return fmt.Errorf("not a git repository")
    }

    // 2. 获取 staged changes
    diff, err := git.GetStagedDiff()
    if err != nil || diff == "" {
        return fmt.Errorf("no staged changes, run 'git add' first")
    }

    // 3. 如果 diff 太大，智能截断
    if len(diff) > 8000 {
        diff = truncateDiff(diff, 8000) // 保留文件头 + 关键变更
    }

    // 4. 获取项目风格参考
    recentCommits, _ := git.GetRecentCommits(5)

    // 5. 构建 prompt
    prompt := buildCommitPrompt(diff, recentCommits, cfg.Commit)

    // 6. 调用 LLM
    spinner := ui.NewSpinner("Generating commit message...")
    spinner.Start()
    result, err := llm.Generate(ctx, prompt)
    spinner.Stop()

    // 7. 交互式展示 + 确认
    fmt.Printf("\n%s\n\n", ui.Bold("Suggested commit message:"))
    fmt.Printf("  %s\n", ui.Green(result.Title))
    if result.Body != "" {
        fmt.Printf("\n  %s\n", result.Body)
    }

    // 8. 用户选择
    choice := ui.Select("Action:", []string{
        "✅ Accept and commit",
        "📝 Edit before committing",
        "🔄 Regenerate",
        "❌ Cancel",
    })

    switch choice {
    case 0:
        return git.DoCommit(result.FullMessage())
    case 1:
        edited := ui.Editor(result.FullMessage())
        return git.DoCommit(edited)
    case 2:
        return runCommit(cmd, args) // 递归重新生成
    default:
        return nil
    }
}
```

#### Review 命令核心

```go
// internal/devkit/cmd/review.go
// 输入: git diff（或指定文件）
// 输出: 结构化 review 意见

type ReviewResult struct {
    Summary    string        `json:"summary"`
    Score      int           `json:"score"`       // 1-10
    Issues     []ReviewIssue `json:"issues"`
    Suggestions []string     `json:"suggestions"`
}

type ReviewIssue struct {
    File     string `json:"file"`
    Line     int    `json:"line"`
    Severity string `json:"severity"`  // error / warning / info
    Message  string `json:"message"`
    Fix      string `json:"fix,omitempty"`  // 建议修复
}

// 输出格式（终端彩色）:
// 📊 Code Review Score: 7/10
//
// 🔴 ERROR api/handler.go:45
//    Missing error handling for database query
//    Fix: Add `if err != nil { return err }`
//
// 🟡 WARNING utils/parse.go:12
//    Potential nil pointer dereference
//
// 💡 Suggestions:
//    1. Consider adding input validation for user-facing APIs
//    2. Add unit tests for the new helper functions
```

---

## 3. 分发与安装

### 3.1 Homebrew Tap

```ruby
# Formula/devkit.rb
class Devkit < Formula
  desc "AI-powered developer CLI toolkit"
  homepage "https://github.com/yourname/devkit"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/.../devkit_darwin_arm64.tar.gz"
      sha256 "..."
    else
      url "https://github.com/.../devkit_darwin_amd64.tar.gz"
      sha256 "..."
    end
  end

  on_linux do
    url "https://github.com/.../devkit_linux_amd64.tar.gz"
    sha256 "..."
  end

  def install
    bin.install "devkit"
  end
end
```

### 3.2 GoReleaser 配置

```yaml
# .goreleaser.yml
builds:
  - main: ./cmd/devkit
    binary: devkit
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}

brews:
  - repository:
      owner: yourname
      name: homebrew-tap
    homepage: "https://github.com/yourname/devkit"
    description: "AI-powered developer CLI toolkit"

archives:
  - format: tar.gz
    name_template: "devkit_{{ .Os }}_{{ .Arch }}"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
```

---

## 4. License 验证机制

```go
// pkg/auth/license.go
type LicenseManager struct {
    keyFile string  // ~/.devkit/license.key
}

func (l *LicenseManager) Validate() (Plan, error) {
    key := l.readKeyFile()
    if key == "" {
        return PlanFree, nil
    }

    // 方案 A：简单 API 验证（推荐 MVP）
    // POST https://api.your-domain.com/license/validate
    // Body: { "key": "xxx", "machine_id": "xxx" }
    // Response: { "valid": true, "plan": "pro", "expires": "2027-01-01" }

    // 方案 B：离线验证（JWT 签名）
    // key 本身是一个 JWT，包含 plan + expiry
    // 用公钥验证签名即可，无需网络请求

    return plan, nil
}

// 在每个命令开头检查
func requirePro(cmd string) error {
    plan, _ := licenseManager.Validate()
    if plan == PlanFree {
        fmt.Printf("⚡ '%s' requires DevKit Pro. Upgrade: https://...\n", cmd)
        return ErrProRequired
    }
    return nil
}
```

---

## 5. 商业化

### 5.1 收费方式

| 方式 | 优点 | 缺点 | 推荐 |
|------|------|------|------|
| **GitHub Sponsors** | 社区认可 | 收入不可预测 | 🟡 辅助 |
| **Paddle/Stripe 订阅** | 稳定收入 | 需要搭建 API | 🟢 主力 |
| **Gumroad 一次性** | 简单 | 无复购 | 🟡 早期 |
| **License Key** | 离线可用 | 需要验证逻辑 | 🟢 主力 |

### 5.2 推广策略

| 阶段 | 动作 | 预期 |
|------|------|------|
| 发布前 | 录制 demo GIF（asciinema） | 吸引眼球 |
| 发布日 | 发 Reddit (r/golang, r/programming) | 100-500 Star |
| 发布日 | 发 Hacker News "Show HN" | 200-1000 Star |
| 第 2 周 | 写 "How I built" 博客 | SEO 流量 |
| 第 3 周 | 提交 Product Hunt | 社区关注 |
| 持续 | 每次发版在 Twitter 发线程 | 持续增长 |
