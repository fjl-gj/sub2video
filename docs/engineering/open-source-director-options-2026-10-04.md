# 开源导演台选型与 Sub2Video 接入边界

核查日期：2026-10-04。范围：上游固定提交的 README、许可证、依赖清单和少量关键源码；没有安装、运行或压测候选项目，没有导入供应商配置，没有访问或修改 107 UAT。

## 结论

有值得借鉴的项目，但没有一个已经核验为可以直接并入 Sub2Video、同时满足现有账户、钱包、任务交付和商业授权要求的完整导演台。

推荐优先研究 **Toonflow 的分镜组织、素材引用、候选结果切换与画布交互**，在现有 Vue 3 / Go 产品内实现一个轻量导演工作区。它的 MIT 许可证和 Vue 技术栈有利于复用，但当前服务没有独立登录鉴权，项目采用本地目录模型，视频节点直接选择上游 provider。不能把整套界面和服务直接接到付费用户入口。

第二梯队按用途选择：**NarratoAI** 用于已有短剧素材的解说和宣传片加工；**MoneyPrinterTurbo** 用于模板化宣传短视频；**OpenCut** 只进入剪辑方案观察名单，当前主仓重写、classic 仓已归档，不作为第一期依赖。

**Huobao Drama** 的功能链最接近完整短剧生产，但当前是 CC BY-NC-SA 4.0，未经书面商业授权不能并入收费产品；单独部署也不会消除非商业限制。**OpenVideo / DesignCombo** 有自定义商用与衍生作品限制，不应按普通 MIT 项目使用。**Storyboarder** 的仓库与网站授权表述不足以确认无限制商业复用。

本轮只形成研究记录。导演台应在当前单镜头主链和代码边界收敛后，以三镜头小样验证，不引入第二套账户、供应商、钱包或任务系统。

## 候选对比

以下“有”表示 README 或源码存在相应设计，不代表本轮完成了运行验收。“剧本→分镜”与“读取已有视频进行解说剪辑”是两种不同能力。

| 项目 | 已核查的主要能力 | 技术和维护证据 | 商业复用边界 | 对 Sub2Video 的判断 |
| --- | --- | --- | --- | --- |
| Toonflow | 同一画布组织文本、角色/场景素材、生成节点和媒体结果；分镜团队示例；3D 导演预演节点 | Vue 3、Pinia、Vue Flow、Element Plus/TDesign；Bun、Express、目录文件；固定提交 2026-10-01，未归档 | 根许可证 MIT；保留版权与许可文本；第三方依赖和媒体另查 | 首选交互/模块研究对象；需替换供应商入口、存储和用户隔离 |
| Huobao Drama | 小说改剧本、角色/场景/道具抽取、分镜拆解、提示词、视频生成、字幕和整集合成 | 当前为 TypeScript、Hono、Drizzle、Mastra、SQLite/Electron；固定提交 2026-10-02 | CC BY-NC-SA 4.0；商业服务和商业二次开发需事先书面许可 | 可作流程对照；未获授权前不复制代码、提示词或资产 |
| NarratoAI | 已有影视素材分析、解说脚本、剪辑、配音、字幕、剪映草稿 | Python ≥3.12、Streamlit、MoviePy；固定提交 2026-09-17，其中最后提交是文档更新 | MIT；素材、声音、模型服务的权利与费用另计 | 适合后续宣传/解说加工；不是角色连续性导演台 |
| MoneyPrinterTurbo | 主题/脚本、素材、配音、字幕、音乐、自动剪辑；WebUI/API/CLI | Python ≥3.11、FastAPI、Streamlit、MoviePy；固定提交 2026-10-03，包含语音流修复 | MIT；外部素材、TTS、模型许可仍需分别确认 | 适合宣传片模板；不能替代镜头版本和选片管理 |
| OpenCut / classic | classic 有时间线、项目和视频编辑；新主仓改用 Rust 核心 | 新主仓固定提交 2026-09-24；classic 固定提交 2026-05-17 且已归档；classic React 19 / Next 16 / Tailwind 4 | 两仓实际 LICENSE 为 MIT；依赖、编解码组件另查 | 可做后续粗剪试验；新主仓 API/headless 等仍属计划，不按现成功能依赖 |
| StoryToolkitAI | 从已有视频/转录中搜索、整理故事，EDL/XML/Fountain 导出，与 DaVinci Resolve 协作 | Python 桌面、CustomTkinter、Whisper；固定提交 2026-07-27 | GPL v3；商业使用不等于禁止，但分发衍生软件有相应义务 | 适合创作者个人后期工具；不适合直接嵌入 Vue 导演台 |
| Storyboarder | 手绘分镜、Fountain 剧本、animatic、桌面导出 | Electron 18 / React 16；默认分支固定提交 2022-06-30；仓库 pushed_at 更新不能代表默认分支仍活跃 | 固定提交未找到根 LICENSE，package 无 license；网站提出“MIT 加例外”，包含收费/分发限制意向 | 用作分镜交互与文件交换参考；商业代码复用暂不通过 |
| OpenVideo / 原 DesignCombo React Video Editor | 时间线、多轨、裁剪/文本/音频、浏览器视频导出 | React/Next 15、Zustand、Tailwind 4、WebCodecs/PixiJS；固定提交 2026-06-30 | 自定义 OpenVideo License；员工规模和衍生作品销售有约束；核心依赖需另查 | 不作为免费商用组件引入；需针对收费 SaaS 的书面许可确认 |

维护日期仅说明所查提交的时间，不能证明可用性、漏洞处理效率、稳定版本质量或未来维护承诺。此表没有把 star 数作为适用性依据。

## 四个值得继续研究的方向

### 1. Toonflow：优先借鉴导演工作区，不整套替换

固定版本可确认：

- `apps/web/package.json` 使用 Vue、Vue Flow、Pinia，但也依赖 Element Plus、TDesign、独立节点运行时和服务器类型。不能直接视为现有 Reka UI / Tailwind 3.4 的组件包。
- `apps/web/src/stores/workspace.ts` 以 `directory` 作为项目身份，浏览器持久化打开过的目录。Sub2Video 的 `projectId`、素材归属和服务端权限不能替换为用户提交的任意文件路径。
- `apps/server/src/routes/workspaces/files/write.ts` 是工作目录文件写入接口。接入需要改为已有素材/项目授权边界，而不是给公网上的每位用户暴露主机工作目录。
- `packages/nodes/videoGenerationNode/src/index.vue` 有媒体历史选择、参考素材、参数和生成动作；请求包含 `providerId`、`modelId`。Sub2Video 客户端只能保留公开承运商/模型与创作参数，供应商和执行方式必须留在后端。
- `packages/teams/storyboardTeam/readme.md` 的导演/编剧/审阅团队是本地示例，明确不自动写工作区、不执行消耗算力的媒体生成。不能宣称已有“一键自动拍完整短剧”。
- README §4.4 明确当前业务页面与 API 没有独立登录鉴权，公网访问需要另设带认证的反向代理。反向代理登录本身也不能证明项目、文件、生成任务已按用户隔离。

值得保留的设计：文本与素材在同一工作区关联、镜头候选结果切换、可见的参考输入、独立的生成动作、画布位置与业务对象分离。第一期不需要照搬节点连线、插件市场、MCP、Agent 工具权限或 3D 导演场景；它们会扩大当前产品的学习和维护成本。

### 2. NarratoAI：作为宣传/解说加工候选

它主要解决“已有影片怎样提炼成解说和宣传素材”。源码有 `clip_video.py`、`generate_narration_script.py`、`jianying_draft_builder.py` 等服务模块。可以研究其片段组织和草稿导出方式，后续按需要封装为内部作业。

本轮没有确认它具有可直接上线的多租户 API。README 中主角人脸匹配、部分自动对应功能仍是未勾选计划，不能作为已完成能力计入 Sub2Video。先用自己的授权素材验证一条 30～60 秒宣传片，再决定是否接入。

### 3. MoneyPrinterTurbo：作为模板化宣传视频候选

它有明确的 FastAPI 启动入口和共享视频处理流程，更适合“给定文案和素材，产出配音、字幕与剪辑”。若未来选用，适合独立的内部渲染进程；前台仍使用 Sub2Video 的项目、作业和结果页面。

不得引入其 provider 设置导入/导出、用户自填 Key、自动发布社交账号等入口。是否采用 Python 处理链，要与较小的 FFmpeg 合成实现做成本比较；若一期只是顺序拼接三个成片，引入整套应用可能比自己实现一个受控合成作业更复杂。

### 4. OpenCut：留待剪辑需求真正成立后

新主仓 README 说明正在重写，Editor API、插件、headless、脚本/MCP 等属于后续方向；classic 有可研究的编辑器，但已归档。classic 的 React/Next/Tailwind 4 也和当前栈不同。

因此第一期不要把“完整在线剪辑器”列成导演台成立的前提。先做镜头排序、选片、简单裁切和粗剪文件导出；若用户确实需要多轨时间线，再比较 OpenCut 的固定可用版本、独立编辑入口以及更小的时间线组件。

## 三种接入方式

| 方式 | 优点 | 当前代价 | 建议 |
| --- | --- | --- | --- |
| 整套 fork 并替换现有产品 | 初期能看到很多界面 | 两套身份/资产/任务/定价；主题冲突；迁移和升级分叉；上游功能存在不等于商业可靠 | 不选。它会放大当前“代码乱”的问题 |
| 独立服务，Sub2Video 调用内部作业 | Python 渲染等职责较独立；可限制资源 | 需要作业鉴权、隔离、取消/重试、结果回传、资源上限；不是简单 iframe | 适用于确有独立运行时需要的后期合成，不用于绕开许可证 |
| 复用数据思路与小模块，重建现有设计系统下的交互 | 保留现有用户链和计费；迁移范围较小 | 必须先厘清版权和模块依赖；需自己建立镜头持久化与测试 | 首选。Toonflow 优先研究这一方式 |

实施规模只可作计划量级：只读比较原型通常是数个工作日；具备服务端持久化、权限、逐镜生成与候选选择的小范围试点，通常是数周；完整 fork、在线多轨编辑和多用户生产级验收容易扩展至数月。以上是范围估计，不是交付承诺；应在三镜头验证后再排期。

## 接到现有业务资产，而不是另造一套生成平台

当前代码入口可核对：

- `frontend/src/api/client.ts`：`estimateTask`、`createTask`、项目/素材和 `saveTaskToProject`。
- `backend/internal/server/router.go`：估价、提交、任务保存路由；旧 quote 仍单独挂载到废弃处理器。
- `backend/internal/handler/client.go`：估价与提交合同。
- `backend/internal/store/direct_routing.go`：路由、消费订单、幂等和事务内任务创建。
- `backend/internal/store/provider_tasks.go`、`tasks.go`：执行状态、交付和退款相关逻辑。
- `backend/internal/store/projects.go`：现有项目与资产边界。

这些是代码定位证据，不构成本轮新增真实交付验收。本轮未修改上述文件。

| 导演台中的对象/动作 | 应接到的现有资产或边界 | 需要补齐的部分 |
| --- | --- | --- |
| 影片工作区 | 现有作品项目的身份、用户归属和资产目录 | 不直接把每个作品目录都当成影片工程；先定义是否在同一项目附加制作信息 |
| 剧本和镜头卡 | 用户输入的剧本/描述 | 持久化镜头顺序、描述、参数版本；自动拆剧本是新增能力，先允许人工维护 |
| 角色/场景参考 | 已授权、属于该用户的素材引用 | 名称、参考图和镜头引用关系；不能承诺模型自动维持人物一致 |
| 单镜头的一次生成 | 现有估价 → 提交 → 消费订单 → Worker → 交付 | 每次生成对应独立任务与请求 ID；超时重试沿用同一请求 ID；主动重做生成新 ID |
| 镜头候选结果 | 已交付的多个任务结果 | 镜头与任务的一对多关系、明确选用哪个结果；候选“未选用”不等于生成失败 |
| 镜头确认/锁定 | 用户的选片决策 | 与生成任务状态分离；不能把“已提交”显示为“镜头完成” |
| 粗剪/导出 | 已有成片资产和对象存储授权 | 版本化的镜头顺序、裁切范围、目标比例；如引入合成作业，单独记录其状态与输出 |

需要新增持久化关系时再决定最小表结构；本研究不先创建工程总线、通用工作流、Agent 平台或抽象渲染服务。

必须保持的业务规则：

1. 每次生成都走正式估价和任务提交合同；不恢复 quoteId，不调用上游生成接口绕过钱包。
2. 估价合计只是当前参考；提交按当时价格计算。批量提交若以后实现，必须限制任务数量和总预算，在后端逐笔或以明确定义的批次规则校验，不以浏览器上的估价总和当作扣费授权上限。
3. Worker 继续使用任务快照，不由导演页面重新挑供应商；选定模型不能静默变更。
4. 角色图、镜头图、结果视频、渲染输入只传可鉴权的资产/任务标识；服务端解析自有存储，不能让浏览器输入任意文件路径或任意上游 URL。
5. 导演台不携带 provider token、API 模型码、workflow 编码、采购成本、上游地址；模型生成、配音、图像生成等新增费用必须另行设计，不能假装已包含。
6. 生成任务、消费订单、退款、镜头编辑状态、粗剪状态分别表达。编辑一个镜头不重新扣费，失败重试不隐式产生新任务。
7. 手机优先完成查看、修改描述、选择参考素材、逐镜生成与选片；复杂时间线/画布按可用宽度简化，不将桌面无限画布原样缩小。

## 建议的最小验证顺序

1. **先收敛当前应用。** 固定本轮代码检查点，解决账户状态、钱包与主创作链的实际问题。导演台不能借扩展功能掩盖既有缺陷。
2. **做三个镜头的可点击原型。** 同一个作品内显示镜头列表、选中镜头的素材/参数、候选成片。人工输入剧本，复用当前三种创作方式，不引入新模型、新供应商或第二套设计系统。
3. **确认数据关系再写持久化。** 用户刷新后仍能找到镜头、参考素材、候选任务和选用结果；并发编辑有明确覆盖/冲突规则；跨用户不能读取或写入。
4. **复用已有成片验证编排。** 先验证排序、选片、保存和导出清单，不必再次付费生成来证明页面连线。新增真实生成仍须明确账号、逐镜参数和预算。
5. **有真实需求再加粗剪。** 首先证明三个已授权视频按顺序合成且音画/时长正确，输出进入自有资产；再考虑字幕、配音、音乐、多轨和第三方编辑器。

第一期退出条件：一个创作者能理解三个镜头的关系，逐镜生成与改稿不会误扣费，刷新后不丢编辑状态，候选可比较、结果可追溯到任务和订单，已选成片能导出。不是“多了一个导演台导航”或“画布可拖动”就算完成。

## 固定来源与许可证证据

所有链接均指向本轮读取的固定提交，除特别说明的网站文章。README 中的能力声明不替代运行验证；MIT 主许可证不覆盖未核验的模型权重、素材、商标和依赖。

| 项目 | 固定提交 | 一手证据 |
| --- | --- | --- |
| Toonflow | `f37b7728f3cedfe6d7d64109a1aba47afafd1da8` | [README / §4.4](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/README.md)、[MIT](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/LICENSE)、[Web 依赖](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/apps/web/package.json)、[项目模型](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/apps/web/src/stores/workspace.ts)、[视频节点](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/packages/nodes/videoGenerationNode/src/index.vue)、[分镜团队边界](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/packages/teams/storyboardTeam/readme.md) |
| Huobao Drama | `f2b4d866e9490f4f71ed4bf14fb7f12648ce1021` | [README](https://github.com/chatfire-AI/huobao-drama/blob/f2b4d866e9490f4f71ed4bf14fb7f12648ce1021/README.md)、[CC BY-NC-SA 4.0](https://github.com/chatfire-AI/huobao-drama/blob/f2b4d866e9490f4f71ed4bf14fb7f12648ce1021/LICENSE) |
| NarratoAI | `9fa69e022d4add41205ee385207561df8796b3f1` | [README](https://github.com/linyqh/NarratoAI/blob/9fa69e022d4add41205ee385207561df8796b3f1/README.md)、[MIT](https://github.com/linyqh/NarratoAI/blob/9fa69e022d4add41205ee385207561df8796b3f1/LICENSE)、[依赖](https://github.com/linyqh/NarratoAI/blob/9fa69e022d4add41205ee385207561df8796b3f1/pyproject.toml) |
| MoneyPrinterTurbo | `01e9eccc1beb7d97cb8819bbae5eef9f0d08866c` | [README](https://github.com/harry0703/MoneyPrinterTurbo/blob/01e9eccc1beb7d97cb8819bbae5eef9f0d08866c/README.md)、[MIT](https://github.com/harry0703/MoneyPrinterTurbo/blob/01e9eccc1beb7d97cb8819bbae5eef9f0d08866c/LICENSE)、[依赖](https://github.com/harry0703/MoneyPrinterTurbo/blob/01e9eccc1beb7d97cb8819bbae5eef9f0d08866c/pyproject.toml)、[API 启动](https://github.com/harry0703/MoneyPrinterTurbo/blob/01e9eccc1beb7d97cb8819bbae5eef9f0d08866c/main.py) |
| OpenCut 新主仓 | `e668010778568641babef2cc40be4703ae6916d6` | [README](https://github.com/OpenCut-app/OpenCut/blob/e668010778568641babef2cc40be4703ae6916d6/README.md)、[MIT](https://github.com/OpenCut-app/OpenCut/blob/e668010778568641babef2cc40be4703ae6916d6/LICENSE) |
| OpenCut classic | `cf5e79e919144200294fb9fed22a222592a0aeea` | [README](https://github.com/OpenCut-app/opencut-classic/blob/cf5e79e919144200294fb9fed22a222592a0aeea/README.md)、[MIT](https://github.com/OpenCut-app/opencut-classic/blob/cf5e79e919144200294fb9fed22a222592a0aeea/LICENSE)、[Web 依赖](https://github.com/OpenCut-app/opencut-classic/blob/cf5e79e919144200294fb9fed22a222592a0aeea/apps/web/package.json)、[仓库归档状态](https://github.com/OpenCut-app/opencut-classic) |
| StoryToolkitAI | `91ab42b11f509aa84bb3801a5a1ba31cf5cf74cb` | [README](https://github.com/octimot/StoryToolkitAI/blob/91ab42b11f509aa84bb3801a5a1ba31cf5cf74cb/README.md)、[GPL v3](https://github.com/octimot/StoryToolkitAI/blob/91ab42b11f509aa84bb3801a5a1ba31cf5cf74cb/LICENSE) |
| Storyboarder | `8b81a25c71d5f7ca46e8d5b8e3d4f7b3968f95c2` | [README](https://github.com/wonderunit/storyboarder/blob/8b81a25c71d5f7ca46e8d5b8e3d4f7b3968f95c2/README.md)、[根目录](https://github.com/wonderunit/storyboarder/tree/8b81a25c71d5f7ca46e8d5b8e3d4f7b3968f95c2)、[package.json](https://github.com/wonderunit/storyboarder/blob/8b81a25c71d5f7ca46e8d5b8e3d4f7b3968f95c2/package.json)、[作者授权观点文章，非固定版本许可证](https://wonderunit.com/thoughts-on-free-and-open-source/) |
| OpenVideo editor | `9a8c5296da4b258f66dfb7ad73de96be62478bca` | [README](https://github.com/openvideodev/react-video-editor/blob/9a8c5296da4b258f66dfb7ad73de96be62478bca/README.md)、[自定义许可证全文](https://github.com/openvideodev/react-video-editor/blob/9a8c5296da4b258f66dfb7ad73de96be62478bca/LICENSE) |

许可证分类应保持准确：MIT 是宽松许可但仍要求保留声明；CC BY-NC-SA 含非商业限制；GPL v3 不等于 AGPL，不因一般网络使用就自动适用 AGPL 的远程交互条款；自定义条款不能仅凭 GitHub 标签或 README 的“开源/免费/SaaS starter”判断可商用。本轮没有选出需要按 Apache-2.0 或 AGPL 接入的候选，也没有完成全部传递依赖审计。

## 本轮验证与未验证

- 已完成：8 个项目族的来源核查，实际许可证阅读，固定提交、维护状态、关键技术栈和接入方式比较；本地生成/项目接口定位。
- 未完成：任何候选的安装、UI 实测、输出视频验证、多用户隔离验收、依赖完整许可证扫描、供应商运行或实际成本验证。
- 网络证据边界：默认本地代理不可达；公开 GitHub/官网读取通过获批的只读网络请求完成。没有发送任何账户、Token、数据库连接信息。
- 修改文件：仅本文；没有引入依赖、迁移、源代码或生成资产。
- 提交/推送/部署：由主任务统一报告；本研究子任务没有执行。
- 临时文件与数据：研究内容在内存中处理，本子任务未创建临时目录、未改业务数据，无清理目标。
- 下一步：在现有代码收敛完成后，先审阅三镜头导演工作区原型和最小数据关系，再决定是否抽取 Toonflow 的具体 MIT 模块。
