# DevKit Suite 商业化阶段 — 详细开发方案 (Development Plan)

> 本文档基于《产品商业化方案》与《前端设计文档》，结合当前 Go 代码库现状，将商业化落地拆解为可操作、可验收的详细工程步骤。
>
> **原则：后端 API 优先，前端并行，分步闭环，优先跑通主流程。**

---

## 一、当前代码库现状盘点

在正式开始前，我们需要认识到目前代码库的状态：

1. **WatchBot（竞品监控）**：有基础的 Cron 抓取（`fetcher`）、Diff（`differ`）、Gemini 分组分析（`GroupChanges`）和邮件/Telegram 推送。但目前**完全没有将结果持久化到数据库中供用户查询**。
2. **NewsBot（AI 热点日报）**：作为全网共用的 AI 资讯情报流（全局抓取、统一提炼简报），设计初衷即是“全局统一筛选呈现”，无需用户配置数据源。目前已有抓取去重，但缺乏前端展示（Web 历史查询）和订阅管理能力。
3. **架构差异**：两者的运行均为独立脚本/守护进程，并没有提供 REST API 让外部调用。

**👉 核心转型（命令行脚本 ➔ SaaS 平台）：**
我们需要引入**关系型数据库**（PostgreSQL 或 SQLite）存储长期的业务状态（用户、组织、竞品配置、分析历史），并架设 **REST API Server** 承接 Next.js 前端的请求。

---

## 二、详细开发步骤：按 Milestone (阶段) 演进

我们将开发周期划分为 4 个主要阶段（对应产品路线图的 4 个 Milestone）。

### 🏈 阶段 1 (Milestone 1)：SaaS 框架搭建与后端重构 (已完成 ✅)

**目标：重构现有引擎，支持 API，支持多用户，跑通基础登录和数据拉取。**

#### 后端 (Go)

1. **[DB] 数据库 Schema 初始化**
   - 建立实体：`users` (账号), `organizations` (团队空间), `subscriptions` (订阅)。
   - 建立业务：`competitors` (竞品), `pages` (受重点监控的独立页面), `snapshots` (快照), `analyses` (分析结果), `news_digests` (日报日志)。
2. **[API] 接入鉴权体系 (Auth)**
   - 使用 Go 自建 JWT 签发与校验，或直接接入 NextAuth/Supabase 接口。
   - 实现端点：`POST /api/auth/register`, `POST /api/auth/login`, `GET /api/users/me`。
3. **[API] WatchBot 引擎数据库改造 (核心)**
   - 修改 `internal/watchbot/pipeline.go`：不再单纯将变更通过邮件发走，而是先写入（Upsert）到 `analyses` 和 `snapshots` 数据库表。
   - 从数据库的 `pages` 表加载每个用户配置的待监控页面列表，取代本地写死或硬编码配置。
4. **[API] WatchBot 与 NewsBot 读取接口**
   - 实现 `GET /api/watchbot/competitors`, `GET /api/watchbot/timeline/:id`。
   - 实现 `GET /api/newsbot/digests`。

#### 前端 (Next.js)

1. **[UI] 工程初始化**
   - 初始化 Next.js 15 (App Router)，引入 Tailwind CSS 4, shadcn/ui, framer-motion。
   - 构建高颜值的 **Landing Page** (响应式，动画交互)。
2. **[UI] 骨架与登录页**
   - 开发 `/login`, `/register` 和鉴权中间件。
   - 实现 Dashboard 的整体 Layout (左侧 Sidebar + 顶部 User Bar)。

---

### 🚀 阶段 2 (Milestone 2)：打造产品 Aha Moment (已完成 ✅)

**目标：打造顺滑的新用户旅程，完善前端核心业务查询视图。**

#### 前端 (Next.js) & 后端 (Go)

1. **[功能] 智能 Onboarding (第一印象)**
   - **前端**：注册后跳转 `/onboarding` 页面，展示"你的行业是什么？"。
   - **后端**：`POST /api/onboarding`，收到用户的选择后，使用**预置行业模板**在数据库直接为用户新建并绑定 3 个竞品（和对应的主干页面）。
2. **[功能] WatchBot - 全局战术中心 (Battlecards)**
   - **前端**：开发全局 `/watchbot` Dashboard，横向展示我方与所有竞品的最新状态和变动活跃度。
   - **后端**：聚合查询接口，获取多竞品的一周活跃度，输出卡片所需数据。
3. **[功能] WatchBot - 时间线与 Diff 详情**
   - **前端**：开发 `/watchbot/competitor/:id`，使用 `lucide-react` 做 Timeline UI。
   - **前端**：关键特性开发：编写基于 React 的 **Diff Viewer コンポーネント**，将旧代码与新代码侧边对比，并侧边悬浮标注了 Gemini 的结构化分析。
4. **[功能] NewsBot - 智能情报流展示**
    - **前端**：`/newsbot` 模块，使用瀑布流或列表展示每日聚合。支持 Markdown 渲染 Gemini 返回的新闻内容。

---

### 💳 阶段 3 (Milestone 3)：防打扰闭环与商业计费 (约 4 讲)

**目标：只收有效的订阅，开始接入收单管道。**

1. **[功能] 智能条件报警 (Smart Alerts) 配置**
    - **后端**：增加 `alert_rules` 表（例如："severity >= high" 或 "contains 'pricing'"）。
    - **后端**：在 `pipeline.go` 的分析末尾增加规则匹配逻辑；若匹配成功，投递任务到邮件或 Webhook (Slack 等)。
    - **前端**：`/settings` 面板，暴露这些选项供用户自主勾选管理。
2. **[计费] Stripe 后端整合**
    - 接入 `stripe-go` SDK。
    - 实现 `POST /api/billing/create-checkout-session`（创建支付链接）。
    - 实现 `POST /api/webhooks/stripe` 处理订阅生命周期回调（如付款成功，将用户 `users.plan` 设为 `pro`）。
3. **[计费] 权限网关 (Gatekeeper)**
    - **后端中间件**：拦截超出 Plan 限制的操作。例如，`Free` 用户调用 `POST /competitors` 时，如已满负荷，则返回 `402 Payment Required`。
4. **[计费] 前端 Pricing 与 Portal (计费门户)**
    - 开发优雅的 `PricingCard` 的订阅版本选择页面。

---

### 🏢 阶段 4 (Milestone 4)：Team Workspaces 与高端企业能力 (约 3 讲)

**目标：增加团队协作粘性，提高企业客单价 (B2B 场景)。**

1. **[多租户] 团队席位管理**
    - **后端**：完善 RBAC 校验。用户操作竞品数据时，强制检查：`当前竞品属于哪个 Organization？当前操作者在此 Organization 里的 Role 是什么？`。
    - **前端**：开发 `/settings/team` 页面，支持邮件邀请团队新成员 (关联已有 `organizations`)。
2. **[协作] Timeline 的讨论功能**
    - **后端**：基于 `analyses` 这个大表的某次异动，建立一张 `comments` 评论附加表。
    - **前端**：在 Diff 视图下，增加评论互动浮层，允许销售团队 tag 成员讨论价格应对策略。
3. **[深度] 大周期 AI 趋势报告 (生成式导出)**
    - 针对 `Pro` 用户，在周末通过 Cron 运行一个大的 Prompt 任务：把本周积累的几十条 diff 浓缩为一篇周度回顾周报，并生成链接展示（或者提供 HTML 导出成 PDF）。

---

## 三、下一阶段动作与环境建议

我们目前正停留在**设计完毕，准备编码的界线**。

**推荐的技术栈路径建立：**

1. **Repository**: 因为我们目前的 Go 代码（后端）全都在 `API-Change-Sentinel` 仓库。推荐直接新建一个 `web/` 或 `frontend/` 目录存放 Next.js 代码，形成 Monorepo 原则，用 `Vercel` 直接构建 `/frontend/` 目录。
2. **Database**: 建议立即使用 `PostgreSQL`（可使用 Supabase 的云端免费 Postgres 或者本地 Docker 启动）取代现有的临时结构。
3. **分阶段执行**：我们先不写任何 UI 和 Stripe代码，**首先着手进行第一步（阶段 1.1 和 1.3）—— 将现有 WatchBot 逻辑的最终结果改写存入关系型数据库中**。

> **是否开始执行？**
> 如果方案可行，我将先执行第一步：**"定义 DevKit 后管所需的关系型数据库 Schema (PostgreSQL)"**。
