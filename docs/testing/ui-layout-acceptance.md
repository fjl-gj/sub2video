# UI 布局与交互验收标准

编制日期：2026-10-06。适用范围：Sub2Video 官网、认证、客户端、导演台和管理端。

本文是可执行检查表，不是全站已完成的验收报告。新增14条几何/键盘合同已执行，结果与修复见 [2026-10-06执行记录](test-process-acceptance-2026-10-06.md)。本表标注的人工和自动化缺口仍未验证，不能因已有测试文件而宣称全部通过。

## 依据与证据边界

唯一视觉合同为 [design-system.md](../ui/design-system.md)。本表把其要求转成操作和断言，不定义第二套配色、组件或布局系统。

| 来源代码 | 依据 |
| --- | --- |
| DS-Surface | design-system.md「One language, two surfaces」的颜色、圆角与表面语义 |
| DS-Layout | 「Typography and layout」「Responsive alignment refinement」的尺寸、边距、容器与排列要求 |
| DS-Create | 「Core page composition / Creation」的输入、估价、预览和状态要求 |
| DS-Assets | 「Overview and assets」「Public tutorials and delivery」的资产与交付关系 |
| DS-Admin | 「Administration」的过滤、记录、分页、弹窗及局部表格滚动 |
| DS-Auth | 「Login and registration」的认证、媒体降级与表单键盘要求 |
| DS-Access | 「Truthful states and accessibility」「Ownership and implementation」的焦点、portal 和组件所有权 |
| DS-Verify | 「Verification」的静态检查、行为测试、审图和证据区分 |
| WCAG | 补充无障碍检查依据：[WCAG 2.2](https://www.w3.org/TR/WCAG22/)，不是新的品牌设计规范，也不代表已完成认证 |

必须分别记录以下证据，不允许互相替代：

- **Mock 浏览器交互**：固定 API 数据下的真实 DOM、键盘、几何和状态切换。
- **视觉回归**：固定字体和数据下的截图差异，经人工审图解释预期变化。
- **真实 API/UAT**：当前发布版本读取到的数据、下载和明确获准的业务操作。
- **人工体验**：真实浏览器缩放、软键盘、触摸、安全区、字体与层次判断。

Mock 页面中显示的任务、余额和成片不是实际生成、计费、退款或对象存储验收证据。

## 执行矩阵

所有宽度均指 CSS viewport，不是显示器物理像素。基准浏览器缩放为 100%；另行执行 200% 缩放。记录浏览器版本、操作系统、设备像素比和字体加载结果。

| 宽度 | 基准高度 | 重点 |
| --- | --- | --- |
| 320 | 844；弹窗另外使用 640 | 最小宽度、主按钮、弹窗内部滚动、底部导航 |
| 390 | 844 | 日常手机排列、移动导航、操作可触达 |
| 768 | 900 | 桌面侧栏出现后，内容容器是否正确降为单栏 |
| 1024 | 900 | 平板/小桌面，侧栏与表单、目录、预览是否争用宽度 |
| 1440 | 900 | 标准桌面，两栏比例与内容轴线 |
| 1920 | 1080 | 大屏内容利用率、1760px 上限与左右平衡 |

新增自动化的六宽度矩阵统一高度为 900；弹窗额外覆盖 320×640 与 1024×900，移动导航为 320/390×844。它不等于上表每个页面、每种高度都已自动化。

人工抽查页面：`/`、`/auth/login`、`/auth/register`、`/app/create`、`/app/overview`、`/app/works`、已有任务详情、`/app/assets` 及作品目录、作品导演台、`/app/wallet`、`/app/tickets`、`/admin/carriers-models`、`/admin/supplier-models`、`/admin/pricing`、`/admin/tasks`。

记录正常、加载中、空数据、错误、禁用、弹窗打开这些状态；没有对应业务证据的状态使用明确标注的 Mock fixture，不能修改真实用户数据来凑截图。

## 布局与尺寸检查表

以下 1–2px 仅用于浏览器子像素舍入，不是截图差异容差。不得放宽断言掩盖布局问题。

| ID | 标准来源 | 量化阈值/通过条件 | 操作与断言 | 自动化文件/人工要求 |
| --- | --- | --- | --- | --- |
| L01 | DS-Layout | document 与 main 的 `scrollWidth - clientWidth ≤ 1px` | 六宽度进入页面，等待内容及字体就绪；记录 main 与 document 的尺寸。表格可局部横向滚动，不能扩大整个页面 | `responsive-layout.contract.spec.ts` 覆盖主要客户端页面；新增 `ui-standards.contract.spec.ts` 覆盖创作、资产及承运商；其余管理页需人工补查 |
| L02 | DS-Layout | 工作区左右剩余边距之差 ≤ 2px；标题组、主内容外框左轴差 ≤ 2px | 在扣除侧栏后的 main 内比较页面 frame；测标题组、完整筛选控件与记录容器，不测前缀图标之后的内部input文本起点；不以整个 viewport 为中心误判侧栏后的内容 | 既有 responsive 测试；新增 UI standards 补管理端标题组、筛选label外框、记录卡片 |
| L03 | DS-Layout | 内容 frame 最大 1760px；移动外侧设计 gutter 16px；桌面 24–40px fluid gutter | 检查 CSS 所有权与 computed padding，再检查几何。滚动条保留宽度需单独扣除，不把 scrollbar 当作设计 gutter | 当前测试检查平衡和宽屏利用率，**未完整自动断言 ceiling/gutter 数值**；人工量测。必要时用 >1920px 的诊断视口触达 1760 上限 |
| L04 | DS-Layout | 1920px 下既有客户端 frame ≥1500px、官网 hero ≥1600px；不能靠隐藏内容维持宽度 | 页面初始状态和长内容状态各检查一次。阈值来自既有项目回归合同，是该视口下的检查下界，不是新的通用组件宽度 | `responsive-layout.contract.spec.ts`；新增 UI standards 加管理端 frame ≥1500px |
| L05 | DS-Layout | 桌面 header 64px、sidebar 200px；手机 header 56px | 测可见 header/sidebar 的 border box；手机不能残留桌面 sidebar 占位；展开 drawer 不是桌面 sidebar，不要求其宽度为200px | **待补自动化**；当前移动导航检查仅验证 drawer 在视口内及关闭行为 |
| L06 | DS-Layout、DS-Create | `workspace` 实际内容宽度 ≤840px 时单栏；>840px 时创作两栏 | 分别令实际容器为 839/840/841px（允许量测舍入，不按 viewport 猜测）；记录 computed grid columns 和两个区域的几何。小屏输入先于预览 | 六视口只能抽样；`studio.contract.spec.ts` 验证小屏输入优先；**精确容器边界尚未自动化** |
| L07 | DS-Create | 两栏时输入列宽 320–420px，预览占余量；单栏时宽度适配容器 | 在 L06 的两栏条件下测列宽；单栏320px视口不能继续强制320px输入列而溢出。不得强制表单与预览等宽 | **精确列宽待补自动化**；既有 responsive 测试检查 frame、媒体居中 |
| L08 | DS-Admin | 七列表格不挤成竖排；滚动局限于表格容器；最右操作列可达 | 在320/390/768px横向滚动表格，检查 `scrollLeft > 0` 且操作列表头落入容器边界；页面本身仍满足L01。宽屏可以不产生横向滚动 | 新增 UI standards 覆盖承运商模型表；供应商/成本/任务表仍需逐页补查 |
| L09 | DS-Assets、DS-Layout | 同级卡片不重叠；卡片落在父容器内；局部目录不挤占内容 | 检查作品卡片、文件卡片、导演台分镜列表及任务预览；在侧栏存在的768/1024px重点观察 | `responsive-layout.contract.spec.ts`、`director.contract.spec.ts`；人工检查实际长标题与空状态 |
| L10 | DS-Layout、DS-Access | 主要触摸操作点击区域宽高 ≥44 CSS px | 测按钮实际可点击元素，不仅测图标或视觉背景。紧凑桌面管理按钮可保持现有规范；手机主要新增/保存操作必须达到44px | 新增 UI standards：创作提交、新建作品六宽度；手机承运商新增、两表面弹窗确认。**不是所有按钮已覆盖** |
| L11 | DS-Layout、DS-Access | 可触达操作落入视口，视口几何仅允许≤1px舍入；普通内容按钮 bottom ≤可见底栏 top；中心与四个内侧四分位点命中自身或子节点 | 自然 `scrollIntoViewIfNeeded` 后 trial click，再用 `elementFromPoint` 检查5个点。1px只用于viewport边缘，不降低44px尺寸、不允许底栏遮挡或命中失败。不能强制点击；portal 弹窗可覆盖底栏，改检查真实 hit-test | 新增 UI standards。真实设备安全区和软键盘仍需人工；本项完整露出是产品要求，不错误宣称 WCAG AA 要求所有控件完全露出 |
| L12 | DS-Layout、DS-Access | 弹窗四边在 viewport 内；内部横向 overflow ≤1px；长内容可纵向滚动到操作 | 320×640打开新建作品与新增承运商；检查默认/宽弹窗。继续人工检查长表单、屏幕旋转和软键盘弹出 | 新增 UI standards 覆盖默认及wide两类；真实软键盘与长表单未自动化 |

## 视觉、键盘与状态检查表

| ID | 标准来源 | 量化阈值/通过条件 | 操作与断言 | 自动化文件/人工要求 |
| --- | --- | --- | --- | --- |
| V01 | DS-Surface、DS-Access | Client canvas/surface 为 `20 22 22` / `27 30 28`；Admin 为 `245 246 244` / `255 255 255` | 检查实际 main 与 portal 弹窗 computed background。期待值引用设计表，不能读取当前 token 再与自己比较 | 新增 UI standards 两表面检查；**同一SPA内跨表面导航后的继承未覆盖**，现有UI无直接跨表面入口，不伪造路由组件内部调用 |
| V02 | DS-Surface | 对应卡片/弹窗12px、控件8px；字体、字重与图标保持既有组件规范 | 检查 shared recipe 与 computed style；页面不得通过自定义class制造另一套按钮。标题桌面25px/500、手机22px；管理页既有24px覆盖若保留需由主任务明确差异 | 新增 UI standards 仅断言弹窗12px和确认按钮8px；完整排版、导航、卡片仍需审图及代码检查 |
| V03 | DS-Access、DS-Verify | 无任意视觉值、`!important` 或 pages/widgets 自建颜色/阴影/按钮recipe | 检查本轮diff和ESLint；feature CSS有独立命名空间；同语义复用shared组件，不因一个页面创建全局variant | `pnpm lint:check` + 人工代码审查；lint通过不代表所有历史样式已合规 |
| V04 | DS-Verify | 截图变化必须有明确预期、原图/新图/差异及人工结论 | 使用固定fixture和本地字体，审桌面/手机，并查看平板/大屏实际图；不得只更新基线或提高容差 | `surfaces.visual.spec.ts`、`journey.visual.spec.ts`、`auth.visual.spec.ts`、`responsive-layout.contract.spec.ts`、`director.contract.spec.ts`；本次新增UI standards不生成新基线 |
| K01 | DS-Access | 打开弹窗后焦点位于弹窗；Tab与Shift+Tab经过全部可聚焦项仍留在弹窗 | 从触发按钮以Enter打开；双向至少各走可聚焦数+1次；不能只检查弹窗可见 | 新增 UI standards，两表面320/1024px；多层弹窗和菜单组合仍需人工 |
| K02 | DS-Access | Escape关闭后焦点回到原触发按钮 | 关闭前不手动聚焦触发器；关闭后直接断言 `toBeFocused`；确认键盘可以继续操作原页面 | 新增 UI standards。关闭按钮、保存成功后的焦点目标如不同，须单独定义合理目标并测试 |
| K03 | DS-Layout | 移动drawer在Escape或导航完成后不存在；backdrop同时移除 | 展开后将焦点放到导航链接，按Escape；重新展开并选择目标页面，验证URL、标题、drawer消失及目标页可操作 | 新增 UI standards：client/admin各320/390px；抽屉完整焦点管理与读屏语义仍需人工 |
| K04 | DS-Access、WCAG 2.1.1/2.4.7 | 所有操作可用键盘；焦点可见；无无法退出的键盘陷阱 | 从地址栏后开始Tab遍历，不用鼠标代替；重点检查素材移除、模式切换、列表、分页、下拉菜单和FAQ | `studio.contract.spec.ts`、`public-home.contract.spec.ts`有局部行为；全站Tab路径与可见focus需人工 |
| S01 | DS-Create | 缺素材、估价等待、估价失败、无供给、余额不足可区分；未知金额不得显示为免费；过期请求不覆盖新参数 | 注入明确的等待/错误fixture，变更参数并逆序返回估价，检查文案、按钮禁用原因与当前金额 | `studio.contract.spec.ts`、`create-availability.contract.spec.ts`；需另用真实API核对业务可用性 |
| S02 | DS-Assets、DS-Create | 只有真实交付URL才出现实际播放器；播放器失败仍可定位任务/下载；保存失败保留输入 | 用无URL、存储等待、播放失败、保存失败fixture，检查不同提示及可恢复操作 | `studio.contract.spec.ts`、delivery/experience-delivery相关合同；下载可播放与持久保存另行真实验收 |
| S03 | DS-Auth | 表单可Enter提交，密码管理器属性正确；错误后保留用户输入；注册受真实公开配置约束 | 检查autocomplete、密码显示、禁用/关闭注册、验证码、条款确认、接口错误、登录回跳 | `auth-forms.contract.spec.ts`、`auth-session.contract.spec.ts`；不证明真实邮件或账号注册成功 |
| S04 | DS-Assets、DS-Access | 导演台未保存更改、冲突、任务候选与镜头关系准确；返回创作不自动提交 | 检查编辑/保存/刷新、冲突保留、候选选择、重新准备、清单导出及注销 | `director.contract.spec.ts`；真实持久化见独立integration记录，本表不借用其历史成功 |

## 需要人工或后续增量自动化的压力场景

| ID | 标准来源 | 量化阈值/通过条件 | 可执行步骤 | 当前覆盖与证据 |
| --- | --- | --- | --- | --- |
| A01 | WCAG 1.4.4；DS-Layout可操作性 | 浏览器200%缩放后信息和操作不丢失、不重叠遮挡；可以自然滚动到全部操作 | 桌面浏览器菜单设为200%，访问创作、资产、导演台、钱包和管理弹窗；记录缩放前后innerWidth。完成后恢复100% | **未自动化，需人工**。`deviceScaleFactor=2`、CSS `zoom` 和截图放大不能替代浏览器200%缩放证据 |
| A02 | DS-Layout、DS-Access；WCAG 1.4.10 | 320 CSS px等效宽度下正文和表单不需要双向滚动；本质二维表格可以在独立容器内滚动 | 320px下核对信息与操作完整，不能只看 `overflow-x:hidden` 后没有滚动条；再用桌面浏览器缩放复核重排 | 六宽度有几何抽样；信息完整性、横向表格例外仍需人工判断 |
| A03 | DS-Layout、DS-Access | 合法长内容不覆盖操作、不产生页面横向溢出；截断文字有可获取的完整内容 | 在隔离fixture中准备64字中文名称、128字符连续ASCII标识和2000字描述；若字段声明更短上限，使用其允许上限并记录。检查列表、卡片、弹窗、错误原因及表单，不把超出业务上限的数据当作UI缺陷 | **未建立系统长文案自动化**；使用fixture不能写入真实业务资产 |
| A04 | DS-Auth、DS-Access；WCAG 2.2.2 | reduced-motion、手机或save-data下背景视频不自动下载/播放；用户明确播放/暂停可用；失败保留封面和认证入口 | 分别设置reduce/no-preference、手机宽度、saveData；统计媒体请求与video状态；切到后台和滚出视口检查暂停 | `public-home.contract.spec.ts`已定义reduce/mobile/saveData合同，auth有播放/失败合同；全站动画与后台暂停需补查，不能把固定reduce截图当作完整检查 |
| A05 | DS-Access；WCAG 1.4.3/1.4.11 | 普通文本≥4.5:1；大文本≥3:1（≥24px常规或≥18.67px粗体）；有意义非文本边界/状态≥3:1 | 用最终渲染色含透明叠加测量正文、辅助说明、按钮、错误、focus与媒体覆盖层；分别测client/admin、默认/hover/focus。禁用及纯装饰按标准例外单独记录 | **未运行对比度审计**；自动工具只做初筛，背景图、透明度和语义例外需人工核实 |
| A06 | DS-Layout、DS-Access | 真机输入时当前字段、错误和确认操作可达；safe-area内不遮挡主操作 | 390px级手机打开创作输入、登录密码和管理弹窗，弹出软键盘、旋转屏幕、滚动到底部；检查底栏与视口变化 | **未验证真机**。桌面Chromium缩窄不证明移动Safari/Chrome、软键盘和刘海安全区行为 |
| A07 | DS-Auth、DS-Access | 200%文本/系统字体放大后字段标签与错误完整；状态不只依赖颜色 | 检查状态同时有文字/可访问名称，使用屏幕阅读器读出模式、输入标签、错误和任务状态；不要强行给每个装饰图标读屏文本 | **待人工**；与浏览器整体缩放不同，不能由A01自动推导 |

## 新增自动化的精确范围

[ui-standards.contract.spec.ts](../../frontend/tests/e2e/ui-standards.contract.spec.ts) 共14条实际用例：

1. 六档宽度各1条：创作提交按钮、新建作品按钮、承运商页面轴线和七列表格。检查尺寸、trial click、5点hit-test、底栏避让与页面overflow；不点击提交。
2. client/admin各2条：320×640和1024×900的弹窗边界、主题色、圆角、双向焦点循环、Escape关闭及焦点归还。
3. client/admin各2条：320/390×844下drawer的Escape关闭与导航后关闭。

测试复用 `fixtures/surface-data.ts`，只追加本文件内的只读认证、目录、选项、承运商fixture。未知业务请求由共享fixture拒绝并记入 `unmatchedRequests`；测试不放行到真实后端，不添加生成、资产写入或配置修改成功的Mock响应。

自定义矩阵在desktop项目运行一次；mobile项目会跳过这14条重复矩阵。全项目发现28条不等于28条独立覆盖。几何断言对单个失败使用soft expect以继续收集同页信息，但该用例仍标为失败；不会通过soft expect忽略缺陷。

尚未由本文件覆盖：所有管理页面、840px精确容器边界、完整字体/gutter测量、200%浏览器缩放、长文本、全站对比度、真实软键盘、移动浏览器差异和SPA跨表面主题继承。对应检查不能记为“已通过”。

## 执行顺序与结果记录

1. 保存当前分支、HEAD、未提交源码摘要和版本；确认字体和固定fixture。不要把旧UAT截图标为当前源码结果。
2. 运行静态检查，再运行本轮新增14条及相关既有交互合同；失败先区分测试错误与产品错误，保留复现证据，不降低标准。
3. 运行视觉回归，人工查看有意修改的桌面/手机基线以及六视口抽样。只有预期变化才更新对应截图。
4. 完成人工压力场景；无法执行的行写“未验证”、具体缺失条件和下一次执行条件。
5. 如果验证真实UAT，单独记录发布版本与数据权限；页面测试不可擅自触发付费生成、修改余额或配置。

从 `frontend/` 运行项目命令：

```text
pnpm lint:check
pnpm typecheck
pnpm exec playwright test tests/e2e/ui-standards.contract.spec.ts --project=desktop
pnpm exec playwright test tests/e2e/responsive-layout.contract.spec.ts tests/e2e/studio.contract.spec.ts tests/e2e/public-home.contract.spec.ts tests/e2e/auth-forms.contract.spec.ts
pnpm test:visual
```

运行前由协调者把本轮报告、trace和失败截图输出到独立 `.tmp_<task>_<id>`，复用已确认的本地测试服务器；不并发启动争用同一端口的实例。若Windows的pnpm shim不可用，使用相同本地依赖的 `node node_modules/@playwright/test/cli.js test ...`，不得借此升级依赖或更换基线环境。

每轮验收记录至少包含：

| 字段 | 必填内容 |
| --- | --- |
| 版本 | 分支、HEAD、工作区差异摘要、构建或开发服务器来源 |
| 环境 | URL、OS/浏览器版本、viewport、缩放、设备像素比、字体 |
| 场景 | 本表ID、页面、状态、可复现操作 |
| 证据类型 | Mock交互 / 视觉 / 真实API / 人工；不得混写 |
| 结果 | 通过 / 失败 / 未验证；附断言或截图路径及解释 |
| 问题 | 实际值、目标值、影响、优先级、负责文件及修复后复测 |
| 清理 | 本轮准确临时路径、是否仍需保留失败trace；不删除历史目录 |

结论只能覆盖实际执行的场景。几何和Mock通过不能推导商业化完成、真实交付成功或封闭Beta通过；这些仍需要业务、数据、安全和实际用户体验的独立证据。
