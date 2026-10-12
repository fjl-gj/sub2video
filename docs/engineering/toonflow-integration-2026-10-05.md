# Toonflow 场景采用与实现边界

核查日期：2026-10-05。上游固定版本：`HBAI-Ltd/Toonflow-app@f37b7728f3cedfe6d7d64109a1aba47afafd1da8`。

## 本次采用什么

用户已选择 Toonflow 作为导演台的参考方案。本次采用其“在同一工作区组织镜头描述与参考素材、查看生成候选、明确选用结果”的工作场景，在 Sub2Video 现有 Vue / Go 技术栈与设计系统内实现。

这不是安装 Toonflow 服务，不是 iframe 嵌入，也不是复用 Toonflow 节点运行时。本次源码研究没有复制上游代码、组件、提示词、商标或媒体资产，没有安装其依赖，没有执行第三方代码。不得把场景借鉴描述成“已集成完整 Toonflow 引擎”或“一键完成整部短剧”。

实施与测试结果由本轮主任务记录；本文件确认来源、许可与接入边界，不替代数据库、浏览器或真实生成验收。

## 固定来源与许可证

本日重新通过公开 GitHub 只读请求读取了下列固定文件，没有携带用户凭据。默认网络代理连接失败后，使用获批的只读网络请求完成核查；未对候选项目执行安装或运行操作。

| 来源 | 核查结论 |
| --- | --- |
| [LICENSE](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/LICENSE) | MIT；版权声明为 `Copyright (c) 2026 HBAI-Ltd`；允许使用、修改和销售软件副本，要求在副本或实质部分中保留版权与许可声明 |
| [README：访问与数据说明](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/README.md#L357) | 第 359 行明确业务页面与 API 没有独立登录鉴权；第 361 行说明数据位于主机目录或 Docker 卷 |
| [README：开源许可](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/README.md#L501) | 第三方依赖和素材另行遵守自身许可；第 507 行保留旧版本的 AGPL / Apache 与附加协议边界，不能把新提交的 MIT 自动套用到旧版代码 |
| [Web 依赖](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/apps/web/package.json) | Vue、Pinia 之外，还依赖 VueFlow、Element Plus、TDesign、多项 workspace 包和节点运行时；不是现有 Reka UI 的可直接替换组件包 |
| [工作区状态](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/apps/web/src/stores/workspace.ts) | 项目以 `directory` 识别，浏览器持久化目录列表；与 Sub2Video 的用户/项目 ID 和资产授权不同 |

许可证中的关键要求为：

> The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

如后续确需抽取实质代码，须记录固定提交、具体文件、本地改动及许可证声明，并检查对应依赖和资产的许可。本轮没有引入这类代码，因此不制造没有实际对应模块的“第三方集成”声明。

## 具体参考的交互

| Toonflow 来源 | 有价值的行为 | Sub2Video 对应场景 |
| --- | --- | --- |
| [视频节点](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/packages/nodes/videoGenerationNode/src/index.vue) | 参考素材、提示词、生成参数、预览和历史结果在一个上下文中关联 | 一个镜头保留描述、素材引用和公开创作参数；生成走现有创作合同 |
| [媒体历史](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/packages/nodeScaffold/src/components/mediaHistory.vue) | 历史列表 → 单候选预览 → 明确“设为当前结果”；区分加载和预览失败 | 镜头关联已有任务，按真实任务状态展示候选，并明确选用成功交付的结果 |
| [引用排序](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/packages/nodeScaffold/src/useNodeReferences.ts) | 引用顺序与内容分离；引用缺失时不假装素材已就绪 | 保留有序素材引用，首尾帧和多图不能混淆顺序；归属与可用性由服务端核验 |
| [画布历史](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/apps/web/src/pages/workspace/panels/canvas/useCanvasHistory.ts) | 编辑结构与生成输出分离 | 编辑镜头或排序不创建新任务、不重新扣费；本期不因此引入无限画布与完整撤销框架 |
| [分镜团队说明](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/packages/teams/storyboardTeam/readme.md#L3) | 文字分镜、生成和审阅有明确边界 | 人工整理镜头与实际生成分别表达；没有真实媒体时不声称成片完成 |

Toonflow 的分镜团队是只读本地示例，说明中明确“不执行消耗算力的媒体生成任务，不自动写入工作区”。不能据此承诺 Sub2Video 已拥有自动编剧、自动拆镜或自动完成整片能力。

## 为什么不嵌入节点运行时

1. **身份与数据归属不同。** Sub2Video 面向多用户，已有鉴权和项目所有权；Toonflow 固定版本以主机目录和本地工作区为主要边界。
2. **计费入口不同。** 上游视频节点直接构造 `providerId`、模型和输出目录并调用生成；Sub2Video 必须使用公开模型合同、幂等提交和既有钱包事务。
3. **界面依赖不同。** 媒体历史组件依赖 Element Plus 和 VueFlow；直接移植会引入另一套控件、主题和节点生命周期。
4. **执行状态不同。** 上游节点的本地 `idle/running/succeeded/failed` 不能替代持久任务、未知提交、存储等待、退款和 Worker 恢复状态。

本次查到最小的无导入 TypeScript 文件是 [values.ts](https://github.com/HBAI-Ltd/Toonflow-app/blob/f37b7728f3cedfe6d7d64109a1aba47afafd1da8/packages/nodeScaffold/src/values.ts)：约 41 行的媒体类型和 `isNodeOutput` 校验。它只校验任意 URL 与 MIME 的结构，不能证明资产归属、任务交付或下载权限，当前没有值得为它增加适配层的业务需要。

`useNodeReferences`、`useNodeGeneration`、`useCanvasHistory` 分别依赖 VueFlow、工具注册与节点事件，均不作为本轮直接复用模块。这里的结论限于本次实际读取的相关源码，不代表上游所有包都不存在可独立复用代码。

## 必须保持的 Sub2Video 合同

- 导演台仍使用已有登录会话、用户、作品项目、素材和生成任务；不建立第二套账户、供应商、钱包或文件服务。
- 浏览器传项目 ID、镜头 ID、素材引用和任务 ID。不得以主机目录、任意上游 URL 或浏览器声称的用户 ID 代替服务端所有权检查。
- 同一镜头可关联多个候选任务，只有归属正确、成功交付的任务可被选为结果。选片是编辑操作，不是再次生成、退款或任务状态变更。
- 镜头描述、顺序和候选关系需要真实持久化；刷新恢复不能仅靠 Toast 或内存数组。多人/多标签编辑应有明确的版本冲突，不静默覆盖。
- 生成仍走 `POST /client/tasks/estimate` 与 `POST /client/tasks`。不生成 quoteId、不绕过钱包、不在打开镜头或回填参数时自动提交。
- 任务幂等检查先于重新路由。新增任务、消费订单、钱包账本及镜头任务关联若属于同一次提交，应在同一事务内一致完成，不能产生已扣费却无法追溯到镜头的半成功状态。
- 任务执行继续使用快照，导演页面不能重新选择供应商。客户端不接收采购成本、供应商凭据、API/workflow、上游模型码或上游地址。
- 素材修改、候选选择、选片顺序与钱包状态分离；失败重试必须有明确的用户动作与幂等边界，不能无限自动发起新的收费任务。
- 已有成片可以证明编排、选片和下载交互，不能被作为本轮新供应商生成成功的证据。

## 测试环境与证据边界

2026-10-05 仅检查了本地环境，没有访问 107：

- 本机 PATH 未发现 Docker、Podman、psql、redis-server、pg_ctl；常见 Docker / PostgreSQL 安装位置也未发现。
- `wsl --list --quiet` 的只读检查退出码为 0，没有返回已安装发行版。
- `SUB2VIDEO_RUN_INTEGRATION`、`SUB2VIDEO_TEST_DATABASE_URL`、`SUB2VIDEO_TEST_REDIS_URL` 均未配置。本次只检查是否存在，不读取或记录连接值。
- 现有 [集成测试辅助器](../../backend/internal/store/integration_test_helpers_test.go) 可复用：要求明确的 test 数据库，创建随机 schema、执行两次迁移、使用独立 Redis probe，结束后清理隔离 schema。
- `backend/docker-compose.yml` 中的数据库名是 `sub2video`，不能未经隔离直接当作本轮测试数据库。
- 现有 Playwright 具备桌面/移动 Mock 与独立真实 API 配置。Mock、截图、Go 单元测试和真实数据库结果须分别报告，不能互相替代。

导演台真实数据库回归至少需要覆盖：跨用户拒绝、已回收项目拒绝、引用不存在/不归属拒绝、版本冲突、并发保存、选用未交付任务拒绝、任务关联与计费原子性、重复提交不重扣。具体通过/跳过结果以本轮执行记录为准。

本文件不承诺完成整片合成、在线剪辑、自动拆剧本、人物一致性、图像生成或新的音频能力。

## 本研究交付状态

结果：已核验固定提交、MIT 文本和关键源码，明确采用导演场景并使用现有栈实现。范围：来源核查、集成边界与本地测试环境检查。修改文件：仅本文。验证：公开源码只读核查和本地配置存在性检查；未运行上游。提交、推送和部署：由主任务统一报告，本子任务未执行。数据影响与清理：未改业务数据，未创建临时文件。风险：完整依赖许可扫描与上游运行验收未做，真实数据库环境仍未具备。下一步：按上述合同审阅本轮原生导演台实现及其验收证据。
