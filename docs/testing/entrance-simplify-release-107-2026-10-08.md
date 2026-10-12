# 入口精简与统一动漫背景 · 107 验收

## 结果与范围

已完成并发布官网、登录、注册入口调整。用户指定的品牌媒体署名、第二行标语、页脚工作台标语和左侧两句补充说明已移除。媒体底部 SVG 改成简洁的文字播放／暂停按钮；认证法律链接与视频控制合并为一行。官网使用与认证页相同的动漫视频和首帧封面，FAQ 同步更正为品牌宣传影像。

保持现有 Vue 与设计系统、全屏 cover 背景、表单合同、移动端封面优先、减弱动态／省流偏好、隐藏暂停和媒体失败回退。未修改后端、数据库、供给、价格或用户资产；没有创建账号、登录会话或付费任务。本记录不是真实供应商生成或商业 Beta 验收。

## 版本与发布

- 分支：`codex/commercial-studio-redesign`，HEAD：`0cafa8d612f4ad8dcebcff58d7d25d674ebd5945`。工作区有既有未提交修改，未覆盖、未暂存、未提交或推送。
- 完整前端包：`619ab991b81e8aee28aee78560a8c1825a1ec999699edbf02115db8eb2dbb02f`。
- 线上 index：`d1945618fe1961dd899364229450eb98c2822ad2f6dd4c6ba6ba50b6bf099cd6`，浏览器实际入口 `/assets/index-D9e-Znx-.js`。
- 视频：`4ecc7aca13fdb3a6cd2079c94632250ff958138f0c0ffcaf788e8833765b5838`；封面：`3ab4555bf085d34c758d698ce5947f9c8e2f2555f5d0292cbb193b7464abefd9`。沿用既有文件，未重新编码或改动原片。
- release：`/opt/sub2video/releases/entrance-simplify-20261008-01/`，含前端包、manifest、修改文件摘要与脚本。
- 使用既有 `deploy/update-107-frontend.sh`，脚本 SHA：`0293916c10cd8b6abe1afb5cc654e2519975b6b063f2fd779b2c1baba4a1368a`。
- 旧前端：`/opt/sub2video/frontend/.dist-pre-20261008220337-36747/`；旧 index SHA：`4cf17a1008c0219ce1dca9e0313fa8561670bf3bed18aa14985f078a573b9dbf`。
- 1062 个旧 hashed assets 检查：缺失 0、内容变化 0。旧页面的懒加载资源保留。
- Nginx 配置检查与 reload 成功；站点配置文件未修改，SHA 仍为 `c6cd9dc6d3b7061f969d2181020b63d5666dabfc69ecec8055734bb409a1a81d`。
- 后端发布前后 PID 均为 4131，服务 active；运行二进制 SHA 仍为 `adc728bf714a33d900b4fed21bfa46f95edd83a99a09559629df0ffeaa8f58e7`。未重启后端、未迁移数据库。
- 没有部署失败或触发回滚。核验了备份 index 与保留资源，未实际演练恢复。

## 修改文件

生产组件与配方：`AuthShell.vue`、`AuthShowcase.vue`、`auth.css`、`ProductHeroFilm.vue`、`ProductHome.vue`、`home.css`，均在 `frontend/src/features/` 下。

文档：`docs/ui/design-system.md`、`docs/ui/auth-entrance.md`、`frontend/public/brand/auth/ATTRIBUTION.md`，以及本记录和下列 107 证据图片。旧海岸媒体与来源许可保留为历史记录，不将其作者或许可套用于动漫品牌素材。

测试：`auth-forms.contract.spec.ts`、`full-background.contract.spec.ts`、`public-home.contract.spec.ts`；`journey.visual.spec.ts` 仅更新注释。更新 8 张预期视觉基线：认证 4 张、官网旅程 2 张、官网响应式 2 张，全部人工审图，没有放宽容差或改动本轮无关基线。

## 验证命令与边界

| 检查 | 当前结果 | 证据边界 |
| --- | --- | --- |
| `pnpm lint:check` | 通过 | 全前端静态检查 |
| `pnpm typecheck` | 通过，构建时执行 | Vue／TypeScript |
| `pnpm build --outDir ../.tmp_entrance_simplify_20261008_01/dist-final` | 通过 | Node v22.16.0、pnpm 10.32.0、Vite 6.4.3 |
| 入口测试与预期基线更新 | 150 passed、14 skipped、0 failed | 浏览器交互、媒体解码、尺寸矩阵、Mock API、视觉 |
| `node node_modules/@playwright/test/cli.js test -c ../.tmp_entrance_simplify_20261008_01/playwright.config.ts` | 440 passed、36 skipped、0 failed、0 flaky，187.5秒 | 完整前端交互／视觉回归；业务 API 为确定性 fixture |
| `git -c core.safecrlf=false diff --check` | 通过 | 文本差异检查 |
| 107 内置浏览器 | 已核验 | 当前线上资源、真实媒体与公开设置 API；未提交真实表单 |
| 后端／数据库／供应商 | 本轮不执行 | 此次仅入口文案与媒体调整，无后端改动，不用前端测试替代相关验收 |

36 个 skip 为既有设计：尺寸矩阵仅在一个项目运行，最小手机表单用例仅在 mobile 运行；对应矩阵已在另一项目执行，不是环境故障或隐藏失败。

## 当前 107 浏览器证据

- 1440 × 900 官网：实际 MP4 解码 `readyState=4`，1344 × 768、9.625秒，时间推进至 1.056秒，muted/loop 为 true。封面与视频均指向同一品牌目录；播放／暂停可操作。
- 1440 × 900 登录：视频解码 `readyState=4`，5 条指定文案均不在 DOM，footer SVG 数量为 0。表单、创建账号入口和法律链接可见。
- 390 × 844 官网、登录、注册：封面填满背景，无横向溢出；重新加载后 video 数量为 0，保持手机默认不请求视频的行为。
- 320px 注册：无横向溢出，输入宽约 239px，页脚自然换行；提交未勾选条款时保持禁用。
- 当前线上注册设置未启用邮箱验证，所以真实注册截图不含验证码；视觉 fixture 明确启用验证码，不能混为真实设置证据。
- 关闭缓存的桌面登录页面读取 `/api/v1/settings/public`：200、application/json、33ms；同期间 MP4：206、video/mp4、70ms，实际收到约 2.06MB。未观察到视频阻塞公开设置返回。这是一次局域网样本，不能推出公网或所有业务接口的性能结论。没有记录响应正文、请求头或凭据。
- 检查结束已恢复缓存与视口设置，关闭 Network 采集；保留官网标签供用户审阅。

实际 107 截图：

- [官网桌面](entrance-simplify-107-home-desktop-2026-10-08.jpg)
- [官网手机](entrance-simplify-107-home-mobile-2026-10-08.jpg)
- [登录桌面](entrance-simplify-107-login-desktop-2026-10-08.jpg)
- [登录手机](entrance-simplify-107-login-mobile-2026-10-08.jpg)
- [注册桌面](entrance-simplify-107-register-desktop-2026-10-08.jpg)
- [注册手机](entrance-simplify-107-register-mobile-2026-10-08.jpg)

## 数据、清理与下一步

业务数据未修改，原视频和既有临时目录未清理。远端 release 与旧前端备份保留。本轮临时目录 `E:\apps\github.com\sbu2video\.tmp_entrance_simplify_20261008_01` 已删除并通过 `Test-Path=false` 核验，删除内容仅为本轮构建、测试和部署诊断文件；本记录、审阅基线和实际107截图保留。

未发现本轮入口调整的阻塞。旧片首尾融合仍存在轻微柔化，未宣称完美无缝；真实注册、发信、登录提交、钱包及生成流程没有在本轮重验。下一步由用户打开 107 官网、登录和注册页审阅精简后的构图与内容。
