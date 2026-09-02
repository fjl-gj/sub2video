# Sub2API → Sub2Video 复用分析

> 对照仓库：`E:\apps\github.com\sub2api\backend`  
> 分析日期：2026-09-02  
> **结论：不建议整体 fork；约 25–35% 平台能力可提取，60%+ 为 API 网关专属应丢弃。**

---

## 1. Sub2API 是什么规模

| 项 | 值 |
| --- | --- |
| 模块 | `github.com/Wei-Shaw/sub2api` |
| 框架 | **Gin** + Google **Wire** DI |
| ORM | **Ent** + 270 个 SQL 迁移 |
| 数据库 | **PostgreSQL**（非 MySQL） |
| 缓存/队列 | **Redis**（重度使用） |
| 代码量粗估 | `internal/service` 500+ 文件，`internal/handler` 135+ 文件 |

它是一个 **AI API 聚合网关**（渠道、账号、代理、API Key、token 计费），不是创作者视频平台。Sub2Video 要的是「广场 → 创作 → 任务 → 作品工程 → 钱包」，两者**用户面有交集，核心域不同**。

---

## 2. 一张图看复用边界

```text
Sub2API 代码库
├── 【可提取】平台层 ~25–35%
│   认证 / 用户 / 支付 / 兑换码 / 公告 / 审计 / 配置 / 中间件 / OSS 抽象
├── 【需改造】混合层 ~10–15%
│   余额字段 / 定价服务 / 订阅 / 邮件 / 工单(无，需新建)
└── 【应丢弃】网关层 ~60–70%
    API Key / channels / accounts / proxies / gateway handlers
    usage_logs / token 计费 / channel monitor / prompt audit / 插件系统
```

---

## 3. 按 Sub2Video 需求逐项对照

### 3.1 高度通用（建议优先提取）

| Sub2Video 需要 | Sub2API 对应 | 复用度 | 关键路径 | 说明 |
| --- | --- | --- | --- | --- |
| **登录/注册** | `auth_service.go`, `auth_handler.go`, `routes/auth.go` | **70–80%** | JWT + Refresh + 邮箱验证码 | 含 OAuth/2FA/Passkey，首期可只拿邮箱密码 |
| **JWT 中间件** | `server/middleware/jwt_auth.go` | **80%** | Bearer 校验、claim 注入 | 需区分 client/admin audience |
| **Refresh Token** | `refresh_token_cache.go` | **75%** | Redis 存哈希、轮换 | 模式可直接搬 |
| **用户资料** | `user_service.go`, `user_handler.go` | **50–60%** | 改密、头像、邮箱 | 去掉 API Key、分组、平台配额 |
| **支付充值** | `internal/payment/`, `payment_service.go`, `payment_order.go` | **60–70%** | 微信/支付宝/Stripe 等 | 订单类型改为「余额充值」 |
| **支付回调** | `payment_webhook_handler.go` | **65%** | Webhook 验签 + 入账 | 与充值商品配置联动 |
| **兑换码** | `redeem_service.go`, `ent/schema/redeem_code.go` | **75%** | Redis 锁 + 防刷 | 与 Sub2Video mock 语义一致 |
| **公告** | `announcement_handler.go` | **70%** | 管理端发、客户端读 | 管理端后期用 |
| **管理审计** | `server/middleware/audit_log.go` | **70%** | `audit_logs` 表 | 管理端操作留痕 |
| **配置/安装** | `config/config.go`, `setup/` | **50%** | Viper + 首次安装向导 | 需删掉 Gateway 巨型配置块 |
| **HTTP 基建** | `pkg/response`, `pkg/logger`, CORS/Recovery | **65%** | 统一响应、zap 日志 | 小文件，复制成本低 |
| **限流** | `middleware/rate_limiter.go` | **70%** | 登录/注册 Redis 限流 | auth 路由已在用 |
| **对象存储抽象** | `image_storage.go`, `repository/image_storage_s3.go` | **50%** | S3 兼容接口 | 改名为 `ObjectStorage`，扩展视频 MIME |

### 3.2 部分通用（提取思路，不能原样用）

| Sub2Video 需要 | Sub2API 对应 | 复用度 | 改造要点 |
| --- | --- | --- | --- |
| **钱包余额** | `users.balance` 列 + `DeductBalance` | **30–40%** | Sub2API 无独立 ledger；Sub2Video 要 `wallet_ledger` 流水表 |
| **定价/报价** | `pricing_service.go`, `video_billing.go` | **20–30%** | 前者是 token/渠道价；后者是 Grok 视频网关价，**不是**「按秒 768P」创作者计价 |
| **订阅** | `subscription_service.go` | **15%** | Sub2Video 明确不做 API 套餐；可忽略 |
| **邮件验证** | EmailService / EmailQueue | **60%** | 注册验证、忘记密码需要 |
| **工单** | 无 | **0%** | Sub2Video 需新建 |
| **作品工程** | 无 | **0%** | `batch_image_*` 是批量生图，不是工程目录 |
| **视频生成任务** | `image_task_handler.go`, gateway | **10%** | 只有「任务+存储」模式可参考，调度逻辑完全不同 |
| **场景广场** | 无 | **0%** | 需新建 `scene_templates` |

### 3.3 不应复用（Sub2Video 产品明确排除）

| 模块 | 原因 |
| --- | --- |
| `routes/gateway.go` + 全部 `openai_*` / `gateway_*` handler | API 网关核心，占 handler 大半 |
| `api_keys`, `channels`, `accounts`, `proxies`, `groups` | 渠道/账号模型，管理端 Provider 层才需要简化版 |
| `usage_logs`, `usage_billing`, `billing_usage_entries` | 按 token 计费，与「按秒视频价」不同 |
| `channel_monitor_*`, `ops_*` | 渠道监控，不是创作者任务监控 |
| `securityaudit` | Prompt 审计，面向网关流量 |
| `plugin_*` | HashiCorp 插件系统 |
| 用户自助 API Key 管理 | 产品方案：客户端不暴露 API Key |

---

## 4. 数量化估算

按**文件/能力域**粗算（不是 LOC）：

| 分类 | 占 Sub2API backend 比例 | Sub2Video 态度 |
| --- | --- | --- |
| 网关 + 上游协议 + 调度 | **~60–70%** | 丢弃 |
| 平台（认证/用户/支付/兑换/审计/配置） | **~20–25%** | **提取** |
| 混合（余额/定价/订阅/存储/Redis 模式） | **~10–15%** | 参考重写 |
| Sub2Video 独有（作品/任务/广场/工单） | **0%（Sub2API 无）** | 新建 |

**可直接节省的开发量：约 4–6 人周**（认证 + 支付 + 兑换 + 中间件 + 配置），若从零写这些。

**不能省的：约 8–12 人周**（视频任务域、作品工程、广场、工单、创作者计费模型、MiniMax 调度）。

---

## 5. 提取策略（三种，推荐 B）

### A. 整体 fork 再删网关 ❌

- 网关代码太多（`openai_gateway_handler.go` 单文件 3500+ 行），剥离成本 > 重写平台层。

### B. 按需复制模块到 `sbu2video/backend` ✅ 推荐

1. 新建 Sub2Video 仓库 `backend/`，**Gin + Wire** 与 Sub2API 对齐（便于对照复制）。
2. **第一批复制**（改 package 名、去网关依赖）：
   - `internal/pkg/response`, `logger`
   - `server/middleware`: jwt, cors, recovery, audit_log
   - `service/auth_service.go` + `refresh_token_cache.go`（精简 OAuth）
   - `service/redeem_service.go`
   - `internal/payment/` 核心接口 + 微信支付
   - `image_storage` → 泛化为 `object_storage`
3. **第二批参考实现**（不复制，按接口重写）：
   - 钱包 ledger（Sub2Video 新表）
   - 视频任务 + Worker
   - 作品工程

### C. 抽公共 Go module `sub2platform` ⏳ 后期

- 两个产品共用时再抽；现阶段复制更快，避免跨仓库版本拉扯。

---

## 6. 数据库差异（重要）

| | Sub2API | Sub2Video 目标 |
| --- | --- | --- |
| 引擎 | **PostgreSQL** | **PostgreSQL**（已统一） |
| 迁移 | 270 个 PG SQL + Ent | 新建精简 PG 迁移；借鉴表结构，不直接跑 Sub2API migrations |
| 用户表 | `users` 含 balance/concurrency/api 相关 | 保留 email/password/role；**去掉** concurrency；加 `workspace` |
| 可直搬的表思路 | `redeem_codes`, `payment_orders`, `payment_audit_logs`, `audit_logs`, `announcements` | 改字段名/枚举即可 |

开发期 SQLite 可以跑，但 **DDL 以 MySQL 为准**；从 Sub2API 搬代码时把 `lib/pq` / Ent PG dialect 改为 MySQL driver（`go-sql-driver/mysql` 或继续 Ent 配 MySQL）。

---

## 7. 与 Sub2Video 菜单的映射

| Sub2Video 客户端 | 从 Sub2API 能拿到什么 |
| --- | --- |
| 登录/注册/验证 | ✅ 认证栈 |
| 首页概览 | ⚠️ 仅用户/余额字段；任务/存储统计需新建 |
| 创作/任务 | ❌ 需新建；可参考 `image_task` 异步模式 |
| 广场 | ❌ 新建 |
| 资产/回收站 | ❌ 新建；OSS 抽象可借鉴 |
| 钱包 | ⚠️ 支付+兑换可搬；ledger 需新建 |
| 工单 | ❌ 新建 |
| 管理端（后期） | ⚠️ 用户管理/订单/兑换码/公告/审计可搬；工作流/Provider 新建 |

---

## 8. 建议的「提取清单」第一批

若下一步开工，建议按此顺序从 Sub2API **阅读并移植**（非整库）：

| 优先级 | 模块 | 源文件 | 预估工作量 |
| --- | --- | --- | --- |
| P0 | 认证 API | `auth_service.go`, `auth_handler.go`, `routes/auth.go`, `jwt_auth.go` | 3–5 天 |
| P0 | 用户表 + 注册登录迁移 | `ent/schema/user.go` → 改 MySQL 版 | 1–2 天 |
| P1 | 支付充值 | `payment_*`, `internal/payment/` | 3–4 天 |
| P1 | 兑换码 | `redeem_service.go` | 1 天 |
| P1 | OSS 上传 | `image_storage.go`, `image_storage_s3.go` | 2 天 |
| P2 | 公告 | `announcement_*` | 1 天 |
| P2 | 管理审计 | `audit_log.go` | 1 天 |
| — | 视频任务/作品/广场 | 无 | 新建 2–3 周 |

---

## 9. 已确认决策（2026-09-03）

1. 数据库：**PostgreSQL**（与 Sub2API 统一）
2. 框架：**Gin**
3. 复用方式：按需提取，不 fork 网关
4. 登录首期：邮箱 + 密码
5. 旧 chi+SQLite 原型：已废弃重建
