# 官网精简版实施与本地验收

日期：2026-10-09。用户批准 `docs/ui/homepage-design-2026-10-09/refined/index.html` 后实施。本记录证明当前工作区的官网页面、前端交互和本地媒体表现，不证明真实登录、钱包、供应商交付或 107 验收。

## 结果与范围

官网 `/` 已调整为三个部分：全幅品牌影像、一个可切换创作模式的产品示意、简洁页脚。首屏只保留“用文字与图片，创作你的下一幕。”与“开始创作”主要操作；移除展开的教程卡、四步流程、FAQ、重复说明及末尾重复创作入口。

正文标题为“从一个镜头开始。”，文生视频、首尾帧、多图在一处切换。桌面展示左侧输入示意与右侧参考画面；手机保留极简描述或素材条、参考图和创作入口，隐藏示意参数。内容沿用既有品牌、字体、色彩和设计系统，没有新增假案例、价格或生成成功状态。

产品示意明确标注“创作界面示意”和“参考画面”。静态提案中的估价按钮替换为真正的“去创作”链接，分别携带 `mode=text`、`first_last`、`multi_image`；官网本身不估价、不提交任务、不读写用户资产。

本轮保留现有 `ProductHeroFilm.vue` 的媒体策略，未修改该文件：桌面延后加载静音视频，手机、减少动态效果和节流用户先看封面，离屏或隐藏时暂停，媒体失败时仍可进入创作。没有新增 MP4、字体或外部 CDN。新增尾帧 JPG 来自已存在的设计资产；用户原始图片未改。

移除官网教程区域后，同时修正教程详情页的旧 `/#guides` 返回地址。正常教程可返回产品首页，未知教程提供现有入门教程 `/guides/coast-letter` 的真实恢复入口；没有新增教程索引路由。

## 本轮修改文件

以下是本轮修改范围，不是整个脏工作区文件清单。既有后端、认证、导演台、创作页及其他快照变更均保留，未整理、暂存或提交。

| 类别 | 文件 |
| --- | --- |
| 官网组成 | `frontend/src/features/guides/ProductHome.vue`、新增 `ProductModeShowcase.vue`、新增 `ProductReferenceImage.vue` |
| 公共导航和恢复入口 | `frontend/src/layouts/PublicLayout.vue`、`frontend/src/features/guides/GuideDetail.vue` |
| 官网样式 | `frontend/src/features/guides/home.css`、`guides.css` |
| 品牌素材 | 新增 `frontend/public/brand/auth/creator-studio-last.jpg`、更新 `ATTRIBUTION.md` |
| 视觉合同 | `docs/ui/design-system.md`、`docs/ui/homepage-design-2026-10-09/refined/README.md` |
| 交互与布局测试 | `public-home.contract.spec.ts`、`full-background.contract.spec.ts`、`responsive-layout.contract.spec.ts`、`journey.contract.spec.ts` |
| 视觉测试 | `journey.visual.spec.ts` 与下面六张经过人工审图的快照 |
| 验收记录 | 本文件及 `homepage-refinement-2026-10-09/` 的七张实际浏览器截图 |

测试文件均位于 `frontend/tests/e2e/`。本轮更新的快照为：

- `journey.visual.spec.ts-snapshots/journey-product-home-desktop-win32.png`
- `journey.visual.spec.ts-snapshots/journey-product-home-mobile-win32.png`
- `journey.visual.spec.ts-snapshots/journey-guide-detail-desktop-win32.png`
- `journey.visual.spec.ts-snapshots/journey-guide-detail-mobile-win32.png`
- `responsive-layout.contract.spec.ts-snapshots/responsive-home-390-desktop-win32.png`
- `responsive-layout.contract.spec.ts-snapshots/responsive-home-1920-desktop-win32.png`

UI/UX 技能用于信息精简、对齐和字号层次；Vue 与 Vue 测试技能用于 Reka 标签页、组件行为与交互验收。所有建议服从现有设计系统，未引入第二套配方或更换技术栈。

## 版本与复核摘要

分支为 `codex/commercial-studio-redesign`，HEAD 为 `0cafa8d`。测试与构建使用未提交工作区，不能仅凭 HEAD 重建本次页面。应用最终改动完成后才执行下述最终检查；其后只补充文档和清理临时输出。

| 文件 | SHA-256 |
| --- | --- |
| `ProductHome.vue` | `d97260b6eba8db81053a5d911f6f639030fc839a74599e5ee8df8d3474d3140f` |
| `ProductModeShowcase.vue` | `b83834c548f205f011f3a40e4177eff37dba5271a89b728654bf59759060502b` |
| `ProductReferenceImage.vue` | `7c799192196ffc0b09740c2af127bfef5704951daa802f0a01c85e88731fc2a5` |
| `GuideDetail.vue` | `f989ad4e6ff3ac1064051abe84bd3c4e5a49a610c34d1a51c770f698e9fa1d7d` |
| `home.css` | `bce18cb553fde0d6b43741c7afc555c8bcb86c092b512326c4b96e5550ac9f96` |
| `guides.css` | `961231484130bf867426ecc87d60d7b0a91422731a0b444643debf5c202b5ef0` |
| `PublicLayout.vue` | `3804ead3968024a27cb6145a2db772167d084df84961a43b64184c45d642415f` |
| `creator-studio-last.jpg`，332,931 bytes | `3b13d72f4f0c736cea0e3e5b6cf2ac449865aa36c6844d37a762baa062758324` |

工具版本：Node `v22.16.0`、pnpm `10.32.0`、Vite `6.4.3`、Playwright `1.62.1`。临时生产构建 `index.html` 摘要为 `9c53d78b2d309513d06eced959eaea98816bd1fa445b9c9a7d184613ffd466f5`；这不是已发布版本或保留的发布包。

## 最终自动化验证

命令在 `frontend/` 执行，差异检查在仓库根目录执行。

| 命令 | 结果 | 证据范围 |
| --- | --- | --- |
| `pnpm lint:check` | exit 0 | 前端静态规范 |
| `pnpm typecheck` | exit 0 | Vue / TypeScript 类型检查 |
| `pnpm build --outDir ../.tmp_home_implementation_20261009_01/reviewed-dist` | exit 0，2476 modules | 当前工作区生产构建，输出仅用于本地验证 |
| `pnpm test:visual '--reporter=list,json' --output=../.tmp_home_implementation_20261009_01/final-results` | **458 passed，36 skipped，0 failed，0 flaky**，共494项 | 当前配置下的交互、布局、视觉及既有前端回归；API fixture 不是真实业务验收 |
| `git diff --check` | 通过 | 差异空白检查；最终文档补充后再次执行 |

JSON 输出变量为 `PLAYWRIGHT_JSON_OUTPUT_NAME=E:/apps/github.com/sbu2video/.tmp_home_implementation_20261009_01/final-results.json`。最终记录起始时间 `2026-10-09T10:24:14.451Z`，执行约244.16秒。

相关交互测试已包含在完整 Playwright 回归中；没有单独声称运行不存在的 Vitest / 单元测试命令。此次不修改后端，未运行 Go、真实数据库或供应商测试。

36个跳过均为重复尺寸矩阵或指定项目适用性，不是基础设施不可用：

| 测试文件 | 跳过数 | 原因 |
| --- | ---: | --- |
| `auth-forms.contract.spec.ts` | 1 | 最小宽度仅在指定项目运行一次 |
| `journey.contract.spec.ts` | 1 | 最小宽度在手机项目运行 |
| `public-home.contract.spec.ts` | 7 | 七宽度矩阵不在两个项目重复执行 |
| `responsive-layout.contract.spec.ts` | 12 | 自定义尺寸矩阵只在桌面项目执行 |
| `studio.contract.spec.ts` | 1 | 390 / 320宽度合同按项目执行 |
| `ui-standards.contract.spec.ts` | 14 | 显式宽度矩阵只执行一次 |

手机项目的实际交互与视觉测试有执行。未扩大快照容差来隐藏布局问题。

前一轮完整运行曾有1项失败：手机旅程测试用 `.first()` 选中了隐藏的桌面教程导航。改为可见的页脚教程链接，随后完整回归通过。一次未加引号的 PowerShell reporter 参数在运行测试前解析失败，修正命令后重新执行；它不是产品失败，也未计入最终结果。

覆盖重点：标签页点击和方向键 / Home / End；只有一个活动面板；三模式素材和参考画面正确；模式经登录返回；支持组合下的既有描述、素材、时长、分辨率及画幅保留；官网不调用私有业务或触发写入；403 / 损坏图片回退；视频错误回退；短屏标题和主要按钮不重叠；教程返回和未知教程恢复。

## 本地浏览器实际检查

使用内置浏览器访问 `http://127.0.0.1:4186/`，对照实际页面检查，没有用静态提案 PNG 充当运行截图。原提案标签保留，实际页面另开标签并标记为可查看结果。

| 检查视口 | 实际检查 | 截图 |
| --- | --- | --- |
| 1440 × 900 | 全幅影像、桌面两栏、对齐、页脚、视频播放与暂停 | [桌面完整页](./homepage-refinement-2026-10-09/desktop-1440.png) |
| 1920 × 1080 | 宽屏字号、前景宽度和影像铺满 | [宽屏完整页](./homepage-refinement-2026-10-09/wide-1920.png) |
| 768 × 1024 | 平板输入 / 预览比例、导航和页脚 | [768平板](./homepage-refinement-2026-10-09/tablet-768.png) |
| 1024 × 768 | 平板横屏和桌面布局边界 | [1024平板](./homepage-refinement-2026-10-09/tablet-1024.png) |
| 390 × 844 | 手机封面优先、素材条、模式切换、创作操作、页脚 | [手机完整页](./homepage-refinement-2026-10-09/mobile-390.png) |
| 320 × 480 | 原生滚动条下的两行标题、按钮触达与短屏 | [320短屏](./homepage-refinement-2026-10-09/short-320.png) |

截图可能包含整页高度，且原生滚动条会减少可用内容宽度；表中是设置的视口，不声称 PNG 画布恰好等于视口。320px视口实际约305px可用内容宽度，因此移除额外内部标题边距；360–389px采用34 / 46px标题，保证两个语义行完整，不拆开“创作”。

实际点击多图入口到达 `/auth/login?redirect=/app/create?mode=multi_image`，登录表单可见，见 [模式回跳截图](./homepage-refinement-2026-10-09/mode-login-redirect.png)。未输入凭据，也未完成真实登录。手机页脚教程、教程返回产品首页均实际点击验证。

本地 Vite 使用故意不可连接的 `VITE_API_PROXY=http://127.0.0.1:9`；官网和本地媒体可用，登录后的服务端业务不在本次验收范围。桌面视频确实播放、暂停，手机初次显示封面。最终已重置浏览器视口覆盖，保留本地预览供用户查看。

## 未验证项与剩余限制

- 未部署107；107当前页面不能作为本次版本验收证据。
- 真实登录、余额、估价、生成、退款、下载、保存作品、数据库和供应商均未验收；不能由此宣称封闭 Beta 或商业上线达标。
- 参考图加载失败有明确回退，切换模式重新挂载或刷新可重试；没有单独的原位重试按钮，是P2改进项。
- 既有创作入口在改变模式时仍可能清空不兼容的模型选择。测试验证支持组合中的描述、素材和主要参数保留，不能据此声称所有旧草稿字段永远不变；本轮没有重构创作页。
- 教程入口使用已存在的入门教程，并非新的教程中心。品牌参考影像不是客户作品或供应商生成证据。
- 只读复核未发现本次官网引入的P0/P1；这不是全系统无P0/P1的结论。

## 数据、清理与下一步

没有修改业务数据、数据库或真实用户资产，没有扣费。没有修改 `backend/data/` 或 `backend/server.linux`。没有创建 / 切换分支，没有暂存、提交、推送、部署或重启107。

本轮临时目录准确路径为 `E:/apps/github.com/sbu2video/.tmp_home_implementation_20261009_01`，只包含此次构建、测试输出和临时报告。协调代理已核对绝对路径、目录无reparse point，独立只读复核了测试JSON；随后删除动作被自动审批以 `blocked by policy` 拒绝，未执行删除，也未换工具重试。该目录仍保留，用户可手动清理，不影响页面运行。没有清理任何旧临时目录。验收汇总及实际截图留在本记录和同名截图目录；原始测试JSON和生产构建不加入Git，必要时可按上述命令重建。

下一步：用户检查本地实际官网的信息密度、排版和模式入口；之后再以明确的发布范围安排107前端更新和实际环境验收。
