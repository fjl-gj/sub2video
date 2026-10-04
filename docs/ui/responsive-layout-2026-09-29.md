# 响应式布局调整与验证记录

2026-09-29。基于 `codex/commercial-studio-redesign` 的未提交工作区，HEAD 为 `a479c330d90b468fa17f2b0b3ce79a3f415b0dae`。保留本轮开始前已有的修改；本记录只描述布局工作，不把其他未提交代码计入本轮成果。

## 结果与范围

参考小米官网的对齐秩序、媒体尺度和简明层次，保留 Sub2Video 现有深色客户端、浅色管理端和暖色品牌。没有更换组件库、添加新配色或修改业务合同。

| 问题 | 调整 |
| --- | --- |
| 官网主视觉、正文、页脚各用不同宽度 | 共用 1760px 最大内容宽度和左右边界；标题及主操作居中 |
| 大屏工作区两侧空白偏多 | 内容使用剩余工作区，24–40px 流动边距；滚动条两侧预留保持居中 |
| 平板保留外侧导航后，多栏内容被挤窄 | 创作、概览、资产目录、交付页按实际容器宽度折叠 |
| 手机操作可能被底部导航遮挡 | 保留 16px 外边距、底部安全区及可滚动的操作区域 |
| 宽屏视频被限高后靠上显示 | 视频在结果容器内垂直居中，保持原始画幅 |

创作左栏在 320–420px 间调整，右栏承载预览；可用内容宽度不足 840px 时改为单栏。对称指共同边界和视觉平衡，表单与预览不强制等宽。

## 本轮文件

- `frontend/src/shared/styles/tokens.css`：布局变量。
- `frontend/src/shared/ui/studio.css`：客户端和管理端共享内容框架。
- `frontend/src/features/guides/guides.css`：官网导航、正文、教程与页脚。
- `frontend/src/features/guides/home.css`：官网主视觉、栏目和移动排版。
- `frontend/src/features/create/workbench/studio.css`：创作栏宽与折叠。
- `frontend/src/features/overview/studio.css`：概览卡片和结果区域。
- `frontend/src/features/assets/library.css`：作品/文件网格和目录折叠。
- `frontend/src/features/delivery/delivery.css`：交付布局与视频居中。
- `frontend/tests/e2e/responsive-layout.contract.spec.ts`：六档宽度的布局合同。
- `docs/ui/design-system.md`：统一布局规范。
- 本记录，以及 surfaces、journey、admin task reconcile 和新增 responsive layout 对应的视觉基线。

以上多个文件在本轮开始前已存在未提交修改；不能把相对 HEAD 的整个差异当作本轮改动。

## 验证

在 Windows、本地新工作区前端执行，未更新快照的最终回归结果：

| 命令（frontend 目录） | 结果 |
| --- | --- |
| `pnpm lint:check` | 通过，退出码 0 |
| `pnpm typecheck` | 通过，退出码 0 |
| `pnpm test:visual --reporter=list --output=../.tmp_layout_20260929/final-verification` | 217 通过，15 跳过，0 失败，2.6 分钟 |
| 仓库根目录 `git -c core.autocrlf=false diff --check` | 通过 |

15 项跳过均为避免浏览器项目重复执行：12 个自定义屏宽用例只在 desktop 项目运行（其中实际包含 320/390 移动宽度）；3 个既有最小移动宽度用例只在 mobile 项目运行。不存在依赖环境缺失而跳过的布局用例。

新增矩阵覆盖 320、390、768、1024、1440、1920px，检查官网共同边界、工作区两侧边距、标题对齐、横向溢出、卡片重叠、三种模式按钮、输入/提交触达、资产目录和视频/下载/保存布局。任务和余额使用确定性响应；播放器使用仓库内已许可的参考影片。未发起真实生成或扣费。

人工复核了官网桌面/手机、创作桌面/手机、概览、教程、素材目录、管理端及宽屏视频详情截图，并在内置浏览器查看本地官网。新增宽屏交付截图首次运行报告缺少基线，已审阅生成图，最终普通回归通过。没有提高截图容差来掩盖问题。

可审阅截图：

- [1920 官网](../../frontend/tests/e2e/responsive-layout.contract.spec.ts-snapshots/responsive-home-1920-desktop-win32.png)
- [390 官网](../../frontend/tests/e2e/responsive-layout.contract.spec.ts-snapshots/responsive-home-390-desktop-win32.png)
- [1920 创作](../../frontend/tests/e2e/responsive-layout.contract.spec.ts-snapshots/responsive-create-1920-desktop-win32.png)
- [390 创作](../../frontend/tests/e2e/responsive-layout.contract.spec.ts-snapshots/responsive-create-390-desktop-win32.png)
- [1920 播放器](../../frontend/tests/e2e/responsive-layout.contract.spec.ts-snapshots/responsive-delivery-1920-desktop-win32.png)

## 边界与后续

本地预览为 `http://127.0.0.1:4186/`，登录后接口仍代理现有 107 UAT。本轮没有部署或重启 107，没有提交或推送，没有数据库迁移或业务数据清理。清理临时测试输出的命令被自动审批策略拒绝，未尝试绕过；仓库根目录及 frontend 下的 `.tmp_layout_20260929` 暂时保留，正式视觉基线也已保留。

当前证据来自 Chromium 和内置浏览器；手机宽度模拟不代表实体 iOS/Safari 验收。容器查询要求现代浏览器。界面回归也不证明供应商交付、钱包正确性或商业上线条件。

下一步：先评审本地首页、创作、资产和任务详情的空间比例，再决定是否发布到 107 供更多设备体验。

## 后续发布到 107 UAT

用户在本轮布局完成后明确授权上传到 107 进行手动体验，以下记录更新上述未部署状态。

- 地址：`http://192.168.0.107:12000/`；创作：`/app/create`。
- 基于当前工作区运行 `pnpm build` 成功，使用现有 `deploy/update-107-frontend.sh` 发布整个前端产物（包括客户端及管理端），并非只上传 CSS。
- 本地与远端归档 SHA-256：`602cc45da71bc268431ece6c0b007ea8dac3fa1a751154854f61001eff7a1ed1`。
- 本地、远端文件和 HTTP 首页的 SHA-256：`c1ea7441bbb3340bcf6737932ea23d46002f57f0c1f36a4498502c1ea0851f2b`。
- 旧版备份：`/opt/sub2video/frontend/.dist-pre-20260929012537`。旧哈希资源保留，避免已有标签页的懒加载资源失效。
- nginx 配置检查及 reload 成功；后端健康检查正常，发布前后 PID 与启动时间一致，没有重启后端或执行迁移。
- 首页、登录、创作、管理端入口 HTTP 200；新入口 JS、全局 CSS、创作及公共页 CSS、海报 MIME 正常；参考视频 Range 请求返回 206。
- 内置浏览器刷新后确认加载新入口 `index-C72E6G3D.js`，首页标题居中、内容宽度变量为 1760px；既有登录会话可读取真实创作目录、余额和任务列表。390px 下创作改单栏，文档无横向溢出。
- 浏览器首次打开仍使用缓存的旧入口，普通刷新后已切换新版。手动体验若看到旧样式，请刷新或使用 Ctrl+F5。
- 没有提交生成任务、上传素材或产生任务费用；没有 Git commit/push。本轮不重新验收供应商交付、钱包结算或管理员恢复操作；仅前端发布不能代表这些后端能力同步升级。

用户选择自行清理，以下临时目录暂留：

- `E:\apps\github.com\sbu2video\.tmp_layout_20260929`
- `E:\apps\github.com\sbu2video\frontend\.tmp_layout_20260929`
- `E:\apps\github.com\sbu2video\.tmp_deploy_layout_20260929`（本次构建归档）
- 107：`/tmp/.tmp_deploy_layout_20260929`（本次部署脚本副本）

这些临时目录不参与服务运行；旧版备份不在清理范围内。下一步是在 107 上手动体验首页、创作、资产和任务详情布局。
