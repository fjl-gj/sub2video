# Sub2Video 测试入口

本目录区分“检查标准”“可执行用例”“本轮运行结果”。历史发布报告不能替代当前工作区验收。

| 文档 | 用途 |
| --- | --- |
| [测试流程和 UI 自动化计划](test-plan-2026-10-06.md) | 环境、数据权限、执行顺序、自动化分层、失败处理和发布门槛 |
| [完整测试用例库](test-cases-2026-10-06.md) | 按业务路径编号的前置条件、操作步骤、预期和现有代码映射 |
| [UI 布局与交互验收标准](ui-layout-acceptance.md) | 六档宽度、对齐、44px 主操作、焦点、遮挡、弹窗、表格与人工检查 |
| [2026-10-06 执行记录](test-process-acceptance-2026-10-06.md) | 本轮真实执行、修复、失败与 skip；这是结果入口 |
| [体验修复候选发布准备](experience-release-preparation-2026-10-06.md) | 发布前的产物、107差异、无需迁移的依据及发布与恢复步骤 |
| [体验修复107发布验收](experience-release-acceptance-2026-10-06.md) | 实际部署SHA、备份、真实只读UI/API结果、数据影响与未验证项 |

视觉合同仍只有 [design-system.md](../ui/design-system.md)。本目录将其转化为测试，不能用于引入另一套页面规范。

## 日常命令

在 `frontend/` 执行，Windows 可使用 `pnpm.cmd`：

```text
pnpm lint:check
pnpm typecheck
pnpm test:ui:interaction
pnpm test:ui:layout
pnpm test:ui:snapshots
pnpm build
```

完整本地浏览器回归使用 `pnpm test:visual`。这个历史命令包含交互合同和截图，并非纯视觉检查。`test:ui:snapshots` 仅选择 `.visual.spec.ts`，部分 `.contract.spec.ts` 内也有截图；发布前仍应运行完整命令。几个入口覆盖有重叠，不需要每次重复执行全部。

Playwright 默认使用隔离 Mock、端口 4185。未知请求的后端代理指向不可用的 loopback 端口，不能意外访问开发者后端。禁止为 Mock 套件设置 `PLAYWRIGHT_REUSE_EXISTING_SERVER=1` 来复用真实 API 代理。测试输出统一指定为本轮 `.tmp_<task>_<id>` 的子目录；使用 `--reporter=dot --output=../.tmp_<task>_<id>/ui` 可避免在旧报告目录写入结果。

在 `backend/` 执行：

```text
go test -count=1 ./...
```

仅执行上述命令、没有集成配置时，数据库测试通常会跳过。真实数据库需要在进程环境中配置 `SUB2VIDEO_RUN_INTEGRATION=1`、`SUB2VIDEO_TEST_DATABASE_URL`、`SUB2VIDEO_TEST_REDIS_URL`，使用专用测试库和 Redis；不要在报告、终端回显或脚本中写入凭据。详细隔离要求见测试计划。

真实 UAT 只读检查：配置 `SUB2VIDEO_LIVE_BASE_URL`、`SUB2VIDEO_LIVE_EMAIL`、`SUB2VIDEO_LIVE_PASSWORD`、`SUB2VIDEO_UAT_PROFILE` 后运行：

```text
pnpm test:uat:readonly --reporter=dot --output=../.tmp_<task>_<id>/uat
```

`SUB2VIDEO_UAT_PROFILE=ready` 要求在真实页面选择文生模式、承运商、模型及5秒/768P/16:9/无自动音频，验证页面实际估价金额和可提交状态，但不点击提交；`unconfigured` 要求明确拒绝生成，证明错误状态与余额不变，不表示生成就绪。缺少配置直接失败，不允许全跳过后报 UAT 通过。此套件不上传素材、不提交任务、不保存作品；新账号、受控写入和付费验收另行按权限执行。

推荐真实只读检查使用上述专用入口。它将认证令牌留在浏览器内，禁用 trace、录像、截图、Service Worker 和 Playwright 1.62 的隐式页面快照，并限制业务写入。旧 `live-*` 套件尚未完成同等的凭据产物加固；仅关闭 trace 不代表失败日志一定安全。升级 Playwright 后须重新执行合成凭据故障验证。

结果统一使用 `PASS / FAIL / SKIP / BLOCKED / NOT RUN`。必须同时报告失败和跳过的原因，不能仅记录命令退出码。
