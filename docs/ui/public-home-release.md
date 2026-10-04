# 官网与认证入口发布记录 · 2026-09-26

## 发布范围

发布到现有 107 UAT：`http://192.168.0.107:12000/`。本次发布前端，包括上一阶段的登录/注册入口与会话修复，以及本阶段的官网改版。未发布后端、执行迁移或修改业务配置；未执行 Git commit/push。

首页从文字与单张配图改为宽幅影像首屏、三种创作入口、分层镜头教程、四步生成交付流程、FAQ 与收尾入口。顶部导航对应创作方式、镜头教程和如何开始。视觉继续使用深色中性背景、暖琥珀主按钮与统一字标。

复用自托管的已授权冰岛实拍视频与海报，明确标注作者、许可和“非生成成片”。手机、减少动态与省流量条件下默认展示海报；视频失败可重试且不阻断创作入口。未添加虚构用户数、价格、客户案例或整片自动制作能力。公开隐私说明更正为当前未开放在线充值支付。

## 主要文件

- `frontend/src/features/guides/ProductHome.vue`、`ProductHeroFilm.vue`、`home.css`：官网内容和媒体行为。
- `frontend/src/layouts/PublicLayout.vue`、`frontend/src/features/guides/guides.css`：公共导航与页脚。
- `frontend/index.html`、`frontend/public/brand/logo-favicon.svg`：页面信息、主题色与图标。
- `frontend/src/pages/auth/PrivacyPage.vue`：支付状态的真实说明。
- `frontend/tests/e2e/public-home.contract.spec.ts`、`journey.visual.spec.ts` 与相关截图：交互和视觉验证。
- `frontend/tests/integration/live-public-entrance.spec.ts`：真实匿名入口、登录回跳和退出验收。
- `deploy/update-107-frontend.sh`：保留旧版 hashed assets，前置 nginx 检查并输出备份位置与校验值，避免已打开页面的懒加载资源在发布后失效。

## 验证证据

- `pnpm lint:check`、`pnpm typecheck`、`pnpm build`、`git diff --check` 通过。
- `pnpm test:visual`：191 通过、3 跳过。跳过项为仅针对手机宽度的测试在 desktop 项目中的重复实例。该套包含 Mock 合同与视觉测试，不作为后端或供应商验收。
- 新增官网合同：24 项通过，覆盖匿名隔离、三模式回跳、草稿保留、FAQ 键盘操作、视频失败重试、减少动态/省流量及 320px 布局。
- 最终 107 版本的真实 `live-creation`：登录、读取目录和三模式估价通过；`SUB2VIDEO_LIVE_CREATE_TASKS=0`，未提交任务。
- 真实 `live-public-entrance`：官网 → 教程指定镜头 → 登录回跳 → 创作目录 → 刷新身份校验 → 退出 → 匿名首页通过。仅允许读取、认证与无扣费估价；测试发现其他写入即拦截并失败。
- 线上 HTML 校验值与本地产物一致；JS、CSS、海报、视频、favicon 的 MIME 正确。视频 Range 请求返回 206；深链、nginx 与后端健康检查正常。
- 内置浏览器检查桌面与手机页面，并确认已有登录会话可以加载工作空间。

## 产物与回退

- 最终 `index.html` SHA-256：`ef7614c9c7282aa50cd598c676b77eb3e75f08203b55caafd4d5dc2b414be4ff`。
- 上传归档 SHA-256：`4d1fc1c7ac2701b5db5ba0c83abfb1731cda69b68e94376bb73492170e1e0347`。
- 官网发布前备份：`/opt/sub2video/frontend/.dist-pre-20260926225055`（已含新版认证入口）。
- 本次首次发布前备份：`/opt/sub2video/frontend/.dist-pre-20260926223701`（本轮完整回退点）。
- [桌面线上截图](./product-home-uat-desktop.png)、[手机线上截图](./product-home-uat-mobile.png)、[首屏截图](./product-home-uat-first-screen.png)。截图来自真实 UAT 匿名页面。

## 验收边界

本次没有创建生成任务、上传素材、注册账户或触发扣费。测试仅产生登录/退出会话与无扣费估价请求。不代表真实支付、供应商稳定性或付费交付已经通过商业验收。

107 仍为普通 HTTP。会话修复覆盖同标签页的并发续期与过期响应保护；跨标签页的 Web Locks 协调需要 HTTPS 或 localhost，不能将本次验收解释为 HTTP 环境下的完整跨标签续期保证。
