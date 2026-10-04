# Sub2Video

面向创作者的单镜头视频生成工作台。当前提供文生视频、首尾帧生成视频、多图生成视频；承运商和模型对用户可见，供应商、成本与执行方式仅在后端和管理端使用。

## 工程入口

| 目录 | 职责 |
| --- | --- |
| `frontend/` | Vue 3 / TypeScript：官网、客户端、管理端；共用设计系统 |
| `backend/` | Go / Gin / PostgreSQL / Redis：身份、路由、账务、任务、Worker、资产 |
| `deploy/` | 候选哈希校验与前后端发布脚本；执行前必须备份、检查迁移和在途任务 |
| `docs/` | 当前业务合同、设计系统、历史验收证据和工程审查 |

正式主链路：准备素材 → 估价 → 幂等提交 → 原子扣费/订单/任务 → Worker 按快照执行 → 交付或明确失败退款 → 下载/保存资产。

预估不生成 quoteId、不扣费、不锁价；首次提交重新计算价格。相同用户和请求 ID 重放直接返回原订单与任务。

## 开始阅读

1. [工程规则](./AGENTS.md)
2. [产品方案总览](./docs/产品方案总览.md)
3. [创作与路由合同](./docs/创作模式与供应商路由设计.md)
4. [设计系统](./docs/ui/design-system.md)
5. [后端配置与隔离集成测试](./backend/README.md)
6. [文档索引](./docs/README.md)

运行配置通过环境变量注入，不提交 `.env`、Token、数据库连接串或业务数据。开发前检查 Git 状态，保留协作者修改。

## 本地验证

在 `frontend/` 执行：

```sh
pnpm install --frozen-lockfile
pnpm lint:check
pnpm typecheck
pnpm build
pnpm test:visual
```

`typecheck` 显式检查 app 与 Vite 配置，不向源码目录输出 JavaScript；Vite 配置源仅为 `vite.config.ts`。`test:visual` 历史命名保留，实际同时包含 Mock 交互合同与视觉回归，不能作为真实账务或供应商验收。运行 Mock 测试时可设 `VITE_API_PROXY=http://127.0.0.1:1`、`PLAYWRIGHT_REUSE_EXISTING_SERVER=0`，阻断未匹配请求访问真实后端。

在 `backend/` 执行 `go test -count=1 ./...`。真实数据库测试默认跳过；需按后端 README 配置专用测试数据库与 Redis。禁止将 UAT/生产连接直接交给测试清理逻辑。

发布脚本的本地 Mock 测试：`bash deploy/tests/update-107.test.sh`，或传入本轮独立 `.tmp_<task>_<id>/deploy` 的绝对路径。`deploy/finish-107-backend.sh` 已废弃，仅返回拒绝执行，不再修改钱包或服务。

单元测试、Mock、视觉、真实数据库、真实供应商和部署验收分别报告。当前代码能构建不等于已经发布，也不等于商业上线就绪。
