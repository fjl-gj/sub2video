# 体验修复候选发布准备（2026-10-06）

候选：`experience-candidate-20261006-01`。目标：107 UAT `http://192.168.0.107:12000/`。本记录对应测试流程建设后的应用更新，**不是再次部署导演台或重置数据库**。

## 结果与发布差异

本轮核对107运行版本、结构和任务状态，构建同一份工作区的Linux后端和完整前端，准备校验清单、只读SQL及发布/恢复步骤。本轮不修改应用源码，不提交、不推送、不部署、不重启，不执行迁移、备份、重置或付费生成。

| 组件 | 107现场基线 | 候选内容 | 依赖与风险 |
| --- | --- | --- | --- |
| 后端 | 20261005导演版，SHA `b8ad0f489bc5b2007898ac6e479ea58cd4088430bb38d3a4e9e8f79b29a8f5d6` | 相对旧候选非测试后端源码，仅`client.go`的无可推荐模型错误映射变化，改为503/SUPPLY_UNAVAILABLE | 不更改路由、计费、Worker或导演合同；必须发布后检查业务响应 |
| 完整前端 | index SHA `88cdf672df2f85dd5243f3093a471418bf4d7ea1bb6f88ef0de5060d9bd8d4fe` | 创作错误分型、指定模型草稿恢复；手机primary按钮最小44px | 包含官网、客户端、管理端；沿用已审图的7张手机基线；发布完整dist |
| 数据库 | 完整001–022且全部checksum与当前文件一致；导演两列、默认值及两个CHECK正确且validated | **无迁移、无重置、无bootstrap** | 若部署时发现结构漂移，应停止重新审阅，不能直接重放SQL |
| 测试与工具 | 前一候选的回归代码 | 新增严格只读入口、交互/布局合同，修正迁移集合及集成超时fixture | 测试文件不编入生产前端；测试通过不证明真实供应商可用 |
| 供给与存储 | 供应商凭据0、存储配置项0；AutoDL/COS环境fallback均未配置 | 本候选不添加、恢复或修改这些配置 | 生成型Beta仍阻塞；本次只能验收清楚的不可用状态 |

107现场：服务active，PID414725，启动于2026-10-05 21:27:11 CST；运行/磁盘二进制SHA一致，HTTP/磁盘index SHA一致。`SUB2VIDEO_SKIP_BOOTSTRAP=true`。unit为`/etc/systemd/system/sub2video-backend.service`，drop-in为`90-skip-bootstrap.conf`和`95-reset-jwt.conf`；未输出环境值或凭据。

Sub2Video站点为`/etc/nginx/conf.d/sub2video-uat.conf`，root为`/opt/sub2video/frontend/dist`，监听12000，配置SHA `c6cd9dc6d3b7061f969d2181020b63d5666dabfc69ecec8055734bb409a1a81d`。PG/Redis容器healthy。当前active、review、WAITING_STORAGE、有效租约、non-terminal及重复成本组均0；2用户、3作品，任务/订单/账本/quote/上传均0。这是只读快照，发布时仍须重新检查。

## 候选身份和产物

分支`codex/commercial-studio-redesign`；HEAD `0cafa8d612f4ad8dcebcff58d7d25d674ebd5945`；保留已有未提交导演台与测试修改。HEAD不能单独识别当前候选。

保留交付目录：`E:\apps\github.com\sbu2video\.tmp_release_20261006_01\candidate`。

| 文件/摘要 | SHA-256 |
| --- | --- |
| `sub2video-server` | `adc728bf714a33d900b4fed21bfa46f95edd83a99a09559629df0ffeaa8f58e7` |
| `sub2video-frontend-dist.tgz` | `ccc8f347933069667dbf1871e15fb73a0d5063a5b9e11a41aadd4c14260729f8` |
| 前端`index.html` | `021594a7a606f580fae7790445b34f8e9179a114e928a8c6ee1772142d58e3f3` |
| `source-files.sha256`，384个构建/测试/部署输入 | `44017d18207944b2c625bc5cece037ded020d06abbf9f89c023c9e3a3d01b33c` |

另包含`manifest.json`、`SHA256SUMS`、两个既有update脚本、`preflight.sql`、`verify-022.sql`、`data-fingerprint.sql`、脱敏`uat-preflight.json`和本文副本。SQL全部只读；verify-022只验证结构，不执行022迁移。data-fingerprint沿用旧只读核验，projects摘要排除导演列，若需要核对镜头内容应单独补全部JSONB摘要；不能把该摘要解释为导演内容全量校验。

源码清单由真实文件字节构成，包含未跟踪输入、公共媒体、快照和部署脚本，按Ordinal排序；不包含`.env`、业务数据、旧临时目录或源码归档。它与上一轮296个代码文件摘要的范围/排序不同，不应直接比较两种摘要。候选不会携带运行环境配置、数据库数据或认证会话；后端仍有既有开发默认值，正式启动继续使用107原有运行配置。

相对与107运行SHA对应的20261005 manifest，384个当前输入中23项不同：19项内容变化、4项新增清单输入、0项删除；详细逐文件清单在manifest。导演台原有实现已在旧版本中，不能全部算本轮新增。

使用保留的生成器核验：

```powershell
./.tmp_release_20261006_01/prepare-candidate.ps1 -Phase verify
```

核验会拒绝源码漂移、载荷hash错误、额外候选文件、重复清单项和目录；源码变化后不可继续部署旧包。`SHA256SUMS`不是数字签名，部署时还要与本记录已知的二进制/归档SHA比较。

## 本轮验证与已有证据

| 检查 | 结果与证据边界 |
| --- | --- |
| 当前测试源码身份 | 296文件摘要仍为`d6d01786988fe179c9ec5cc70519ceecdaf6dbd4a3f9b6c7314b0736a9b9f3fd`，与[上一轮实际记录](test-process-acceptance-2026-10-06.md)完全一致；本轮没有修改应用或测试代码 |
| Linux后端交叉编译 | PASS；Go1.25.1，`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o ../.tmp_release_20261006_01/candidate/sub2video-server ./cmd/server`；校验ELF64/x86-64及`vcs.modified=true` |
| 完整前端构建 | `pnpm.cmd build --outDir ../.tmp_release_20261006_01/work/frontend-dist` PASS，包含独立typecheck；Vite6.4.3，2472模块，15.84秒；index与上一轮同源码构建SHA一致 |
| 发布脚本 | bash语法检查＋`bash deploy/tests/update-107.test.sh /e/apps/github.com/sbu2video/.tmp_release_20261006_01/deploy`：12 PASS；systemctl/nginx/curl全为Mock，没有部署 |
| 归档 | PASS；133文件逐项解包SHA一致；无环境文件、源码map、日志、密钥或reparse point，归档路径安全。后端约23.12MB，前端归档约3.39MB |
| 107只读 | 运行SHA、服务、22迁移checksum、结构、任务状态、配置存在性；SQL为READ ONLY，未取出供应商密码或连接串 |
| 上一轮同源码检查 | lint/typecheck/build、Go回归；浏览器341 PASS/29明确去重SKIP；store38、handler26及迁移/Worker故障真实DB证据。**本轮没有重跑这些完整套件** |
| 本轮未执行 | 新候选在107的业务验收、真实存储往返、供应商交付、可播放下载、受控写入、真机；不引用旧成片当作本候选交付 |

构建与包装成功不等于可以无条件开放生成。本候选具备进入短维护窗口内受控文件发布的条件，发布前实时门禁、切换后业务检查仍须执行；没有在当前轮重跑真实供应商连通性测试。

前端构建不读取运行凭据；核查未发现前端`.env*`文件或VITE_*进程变量，源码未使用import.meta.env注入业务秘密。本轮所有工具日志最终只保留文字结果，不进入发布包或Git。

## 准确发布顺序（本轮未执行）

1. 固定候选，核对上述SHA、完整`SHA256SUMS`、源码与manifest。远端使用新目录`/opt/sub2video/releases/experience-candidate-20261006-01/`，权限0700、umask077，不覆盖旧release。上传后重新校验。
2. 发布前确认运行基线仍为b8ad0f48版，22迁移checksum未变，bootstrap仍禁用；出现变化立即停止重审。**不执行migrate-only、seed、数据库reset或历史重建脚本。**
3. 在Sub2Video站点建立短维护窗口并确认实际HTTP生效，阻止新任务/业务写入。**必须保留受控验收通道**：本机127.0.0.1与明确指定测试来源可读取静态页面/GET/HEAD，并仅允许POST到登录、refresh和estimate；任务创建、上传、作品/配置写入等对测试来源也继续拒绝。基于实际连接来源限制，不能信任任意请求头来绕过门禁。前端脚本固定探测`http://127.0.0.1:12000/`，该通道必须返回真实index，不能统一503。reload后状态切换可能短暂延迟，仅做有界探针；先验证本机首页可读、外部业务写入被503拒绝（用无认证空请求，避免触发真实操作），无法建立通道就停止，不能先发布后临时解除门禁。不得修改其他站点。门禁生效后再次确认所有在途/核对/存储等待任务为0。Worker随API运行且没有完成等待屏障，不能把stop称作平滑排空。
4. 保留**此时实际运行版本**的二进制、完整dist和Sub2Video站点配置，逐项SHA；不要误选20261005之前的`.pre-*`作为回滚目标。本次不改schema或业务记录，不需要迁移前dump或数据库恢复步骤；如发布现场需要额外备份，按明确范围单独执行，不以恢复dump作为应用回退。
5. 保持原运行环境，执行候选`update-107-api.sh`，默认`SUB2VIDEO_ROLLBACK_COMPATIBLE=false`。核对新运行SHA、active、登录/目录/任务/钱包及无供给估价503。Worker一同重启；`/health`只能作存活检查。
6. 后端业务检查后执行`update-107-frontend.sh`发布完整dist。归档已预先校验普通文件和安全路径；保留旧hashed assets，验证HTTP index SHA、官网/登录/客户端/管理端入口及懒加载。两个rename之间存在短窗口，在维护状态下操作。不要清理旧chunks。
7. 在受控访问下执行下节无扣费验收，再恢复原站点配置和访问；核对原配置SHA。若当前仍无供给，门槛是错误表达准确、免费导演准备可用，不能称为生成Beta已通过。不得为测试临时修改供给、价格、余额或发起付费任务。

只有完成上述前置检查后才执行，不能单独粘贴即视作完成发布：

```bash
release_dir=/opt/sub2video/releases/experience-candidate-20261006-01
SUB2VIDEO_API_CANDIDATE="$release_dir/sub2video-server" \
  bash "$release_dir/update-107-api.sh" adc728bf714a33d900b4fed21bfa46f95edd83a99a09559629df0ffeaa8f58e7
SUB2VIDEO_FRONTEND_ARCHIVE="$release_dir/sub2video-frontend-dist.tgz" \
  bash "$release_dir/update-107-frontend.sh" ccc8f347933069667dbf1871e15fb73a0d5063a5b9e11a41aadd4c14260729f8
```

## 失败与恢复

| 失败 | 处理 |
| --- | --- |
| SHA/结构/运行身份不符或在途任务出现 | 不切换；重审差异/任务状态。未知提交不得直接退款或重提交 |
| 后端启动/health检查失败 | 保持维护；脚本默认停止候选；查明原因后，按已确认同schema条件恢复本次保存的b8ad0f48二进制和配套前端。恢复会带回旧500提示，属于应急恢复，不是功能验收通过 |
| health通过但真实接口不满足合同 | 脚本不会自动识别或停服务；发布操作者必须保持门禁、显式停止候选并确认进程退出，再按同schema和正确SHA决定人工恢复。不能带着失败结果开放任务写入 |
| 前端入口或reload失败 | 既有脚本尝试恢复旧dist并保留新chunks；验证旧HTTP index及页面；不得无界重试发布 |
| 出现新订单、上游受理或资产写入后失败 | 停止新增写入、保存现状和事件，优先前向修复；不恢复旧数据库覆盖记录 |

本次回滚目标已经含导演合同和022；不沿用更早021版本可能忽略directorContext的回滚前提。仍需现场核对真正备份SHA及启动环境，不能用同Git HEAD推定兼容。服务恢复和业务恢复分别验证。

## 发布后验收清单

1. 确认运行二进制、HTTP index和当前数据库版本；旧打开页面懒加载无404。
2. 登录回跳、三种模式入口、承运商/模型状态、已有草稿描述/显式模型保存正确。
3. 无供给时按钮禁用、提示可理解；合法5秒无扣费估价返回503/SUPPLY_UNAVAILABLE；不再提示成网络故障。若管理员已另行配置供给则使用ready profile，不能把真实环境变化当测试失败后随意放宽断言。
4. 在`frontend/`使用进程内账号配置执行`pnpm test:uat:readonly --reporter=dot --output=../.tmp_<run>/uat`；当前预期profile为unconfigured。仅认证/读取/estimate，不上传或提交。对比余额、任务、流水；订单/quote完整无副作用另由DB只读核对。
5. 320/390/768/1024/1440/1920检查无页面横溢、主要按钮可触达、弹窗与管理表格正常；已有本地Mock/视觉不代替已部署页面检查。
6. 已有作品和导演板只读打开正常；本次没有可用真实成片，下载播放/失败退款必须列未验证。受控写入须另定专用账号与对象范围；再次创作不能自动提交。

## 数据影响、保留与下一步

保留candidate及根目录`prepare-candidate.ps1`作为待发布交付物，均被Git忽略。已核对绝对路径及无reparse point，删除本轮`E:\apps\github.com\sbu2video\.tmp_release_20261006_01\work`和同级`deploy`：共360个构建、解包检查、探针和Mock临时文件，约7.88MB，未使用回收站；它们可按本记录重新生成。两目录不存在已复核，清理后候选校验仍通过。历史`.tmp_*`不动。`backend/data/`、`backend/server.linux`未修改；无暂存文件、Git提交或推送。

本轮可纳入Git的增量仅本文和`docs/testing/README.md`入口链接；应用、测试及部署脚本均未新增修改。`git diff --check`通过；上述两个受保护路径差异为0。候选与验证脚本是待发布交付物，不能在发布前按临时日志清理。

当前候选的作用是交付已验证的最小体验修复；供给/存储缺失仍是生成型试用P1。下一步发布范围仅为这份同schema前后端、原版本文件保留、短维护窗口及无扣费验收；不包含数据库重置、供应商配置、余额修改、付费生成或Git推送。
