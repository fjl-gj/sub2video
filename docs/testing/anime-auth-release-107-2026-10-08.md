# 动漫认证背景发布107与网络验收 · 2026-10-08

## 结果与范围

用户授权“推上107，然后看视频加载会不会影响数据返回”。完整前端候选 `authanime-20261008-01` 已于北京时间20:23:01发布到 `http://192.168.0.107:12000/`；登录与注册均使用前一轮制作的动漫静音循环视频及封面。官网仍使用原海岸参考素材。

正常局域网与1Mbps限速的真实浏览器对照未发现接口等待视频完成的依赖。弱网下视频尚未就绪时公开设置接口已经返回。数据包共用带宽的风险仍存在，不能由一个小响应接口推断所有任务、钱包和上传接口都没有性能影响。

本轮没有修改应用源码、后端、数据库、Nginx站点内容或用户业务数据，没有提交或推送Git。新增本记录、脱敏网络JSON及3张实际107截图。使用已有部署脚本轮换前端并reload Nginx；旧前端与候选继续保留。既有用户和协作者改动完整保留。

## 构建、候选与版本

分支 `codex/commercial-studio-redesign`；HEAD `0cafa8d612f4ad8dcebcff58d7d25d674ebd5945`。工作区已有未提交及未跟踪源码；候选以172项实际构建输入摘要标识，不以HEAD宣称纯提交构建。发布后再次核对这172项输入，变化数为0。

Node `v22.16.0`、pnpm `10.32.0`、Vite `6.4.3`。构建命令：`pnpm build --outDir ../.tmp_authrelease_20261008_01/dist`，包含typecheck，2472模块构建成功。归档仅包含构建目录，没有环境文件、凭据、业务数据或其他临时目录。

| 对象 | SHA-256 |
| --- | --- |
| 完整前端归档 | `7578a9b508a6c4564ed6b98a3625394067d8d89b7fa68ff31dacd4e56c3135a3` |
| 新index.html（磁盘与HTTP均验证） | `4cf17a1008c0219ce1dca9e0313fa8561670bf3bed18aa14985f078a573b9dbf` |
| 静音MP4（源码、磁盘与HTTP均验证） | `4ecc7aca13fdb3a6cd2079c94632250ff958138f0c0ffcaf788e8833765b5838` |
| 封面（源码、磁盘与HTTP均验证） | `3ab4555bf085d34c758d698ce5947f9c8e2f2555f5d0292cbb193b7464abefd9` |
| 既有部署脚本 | `0293916c10cd8b6abe1afb5cc654e2519975b6b063f2fd779b2c1baba4a1368a` |
| 172项构建输入清单 | `14e5ceb3d4391a19d1ab34d9847f23976056d88700c02e34b31ca2a27e4c5b99` |
| 脱敏网络记录JSON | `6e0416649053e5d53ff1cdac70ae0f2bd052905462630317a380775a71ae4127` |
| 发布前index.html | `e43b4b4e81d3161d05c664a65b31ae435510978df3d0e8112ec18971f4b42fac` |

前端归档5,673,950字节。视频2,064,807字节、9.625秒、1344×768、24fps、无音轨；封面218,311字节。媒体生成及轻微融合柔化的限制见 [本地接入记录](anime-auth-background-2026-10-08.md)。

## 发布与回滚材料

- 独立发布目录：`/opt/sub2video/releases/authanime-20261008-01/`，0700权限，保留归档、manifest、源文件摘要、SHA256SUMS及旧Nginx站点副本。
- 旧前端：`/opt/sub2video/frontend/.dist-pre-20261008202301-439608/`。旧index SHA与发布前一致。
- 当前前端：`/opt/sub2video/frontend/dist/`。
- Nginx实际站点：`/etc/nginx/conf.d/sub2video-uat.conf`，发布前后SHA均为 `c6cd9dc6d3b7061f969d2181020b63d5666dabfc69ecec8055734bb409a1a81d`。
- 后端发布前后：PID `1266563`，启动时间 `2026-10-06 11:46:39 CST`，active；运行二进制SHA均为 `adc728bf714a33d900b4fed21bfa46f95edd83a99a09559629df0ffeaa8f58e7`。
- 旧dist内955项hashed assets逐项核对：缺失0、内容变化0，已经打开的页面可继续请求旧懒加载资源。

顺序：只读核查 → 构建／定向回归 → 独立目录与站点备份 → 上传 → 校验归档、脚本、索引、媒体和源摘要 → 确认旧入口未被并行部署替换 → 既有 `update-107-frontend.sh` 切换 → `nginx -t`、reload与HTTP索引验证 → 浏览器真实验收与网络测量 → HTTP媒体摘要、Range及后端不变复核。

发布命令：

```bash
SUB2VIDEO_FRONTEND_ARCHIVE=/opt/sub2video/releases/authanime-20261008-01/sub2video-frontend-dist.tgz \
bash /opt/sub2video/releases/authanime-20261008-01/update-107-frontend.sh \
7578a9b508a6c4564ed6b98a3625394067d8d89b7fa68ff31dacd4e56c3135a3
```

未触发回滚。既有脚本的失败恢复会恢复旧dist并保留新hashed assets；本轮未执行恢复演练。需要人工恢复时应使用相同部署锁，在独立staging准备上述旧前端与当前hashed assets，再轮换、验证Nginx与旧入口SHA，不直接删除当前或历史发布。不存在数据库迁移或数据库回滚需求。

## 真实107网络测量

通过内置浏览器CDP读取请求时序，目标为当前已发布107，不使用API Mock。桌面1440×900，禁用缓存；对照仅改变浏览器减少动态偏好：`reduce`为只封面，`no-preference`为自动播放。慢网络模拟为150ms延迟、下载125,000字节/秒（1Mbps）、上传62,500字节/秒。

量测API为 `GET /api/v1/settings/public`，真实响应84字节JSON，200；含HTTP头传输453字节。所有计入样本均捕获当前文档loader的请求与loadingFinished，不把旧DOM仍可见或请求仅发出当作完成。

| 网络／显示方式 | 三轮API总耗时（ms） | 中位数（ms） |
| --- | --- | --- |
| 正常局域网／自动播放 | 14、18、26 | 18 |
| 正常局域网／只封面 | 26、15、21 | 21 |
| 1Mbps／自动播放 | 178、235、209 | 209 |
| 1Mbps／只封面 | 187、233、175 | 187 |

两组样本范围重叠。慢网中位差22ms不构成统计上确定的性能退化归因，也不代表毫无影响；此小样本只能说明没有等待整段视频的明显阻塞。后台网络、字体和脚本同样可能贡献波动。

更直接的证据：1Mbps自动播放第一轮，API开始于导航后3173ms、完成于3351ms；视频开始于3179ms，此时仍未完成，视频元素 `readyState=0`。另外两轮API也在视频未就绪时完成。正常网初次实测API17ms完成、视频57ms完成；视频通过原生206 Range下载，与API分别处理。

源码因果检查：认证媒体的 `nextTick`／`video.play()` 仅由AuthShowcase内部等待，挂载使用 `void play()`；公开设置、登录和注册的API请求不等待这个Promise，也不经过媒体请求层。`preload=none`不阻止手动／自动play触发下载。当前桌面自动播放在mounted触发，并没有实现全页面load／idle延后；本轮没有把建议优化偷偷作为已实现结果。

同源仍可能争用带宽；实际协议HTTP/1.1。不能从这次小配置响应推断更大的任务列表、钱包数据或上传在各种弱网与并发情况下完全没有影响。真机CPU／GPU解码压力也未量测。若后续出现明确卡顿，可采用官网已有load／idle启动方式再对照验证，不需要增加新的请求抽象。

详尽14项脱敏记录见 [网络样本JSON](anime-auth-107-network-2026-10-08.json)。包含12个对照样本、1个手机样本和1个离开认证页样本；没有请求头、响应业务正文、Token或凭据。

初始诊断曾在页面／请求未完成时过早取样，以及在大量dataReceived事件下丢失缓冲旧事件。这些样本均未计入上述表格；改为及时采集关键事件、匹配当前loader并等到API完成，最终样本无truncated。TTFB使用response.timing的网络头时点，不以可能迟到的responseReceived事件分发时点替代。一次等待裸注册URL因真实回跳query而超时；刷新AX确认实际已进入 `/auth/register?redirect=/app/overview`，不是注册跳转失败。

## 页面与退出行为

- 桌面新版脚本 `/assets/index-B_N-0K1N.js` 与动漫背景可见，真实MP4 `readyState=4`、duration9.625、muted，实际播放正常。
- 手机390×844、允许动态：初始视频元素0、MP4请求0，只请求封面；公开设置16ms完成。登录和注册切换保留封面，注册依据真实公开设置呈现，不增加假邮箱验证码／找回密码。
- 限速下离开认证页到隐私页：视频元素变为0，原视频请求出现 `net::ERR_ABORTED`、`canceled=true`，没有拖入后续页面继续下载；API已于196ms完成。这证明本次浏览器会取消未完成请求，不表示所有浏览器实现都已经验证。
- 视频Range独立HTTP检查：`206 Partial Content`、`Content-Type: video/mp4`、1024字节、`Content-Range: bytes 0-1023/2064807`。
- 所有缓存禁用、网络限速、媒体偏好和视口覆盖在检查结束后恢复；保留107登录页作为交付。

实际107截图：[桌面登录](anime-auth-107-login-desktop-2026-10-08.jpg)、[手机登录](anime-auth-107-login-mobile-2026-10-08.jpg)、[手机注册](anime-auth-107-register-mobile-2026-10-08.jpg)。此前本地截图不用于替代本次107验收。

## 验证命令与证据边界

| 检查 | 本轮结果 |
| --- | --- |
| `pnpm lint:check` | 通过 |
| `pnpm build --outDir ../.tmp_authrelease_20261008_01/dist` | 包含 `pnpm typecheck`，构建通过 |
| Node直接运行Playwright，认证表单／认证视觉／全背景三个spec | 105 passed、1重复最小宽度场景skip、0 unexpected／flaky，46.3秒；Mock业务、真实本地媒体 |
| 既有4张认证视觉基线 | 正常比较通过，未更新或提高容差 |
| `git -c core.safecrlf=false diff --check` | 通过；新文本另做独立检查 |
| 107服务与HTTP文件验证 | 后端不变、Nginx配置不变、索引／视频／封面SHA匹配、955旧资源保留 |
| 107内置浏览器网络检查 | 真实公开设置及媒体；12个对照样本，手机／路由退出另验，无业务写入 |

没有运行Go／数据库集成／真实供应商测试：本轮不改后端，也不以这些测试证明背景性能。没有输入真实认证凭据、创建账号、生成任务、扣费或保存资产。真实登录POST及登录后钱包／任务／目录接口未量测；不能把前端Mock的登录成功或本次公开设置GET当成这些接口的真实环境验收。真机Safari、软键盘和低性能设备解码仍未验证。

## 数据影响、清理与一个下一步

业务数据无本轮写入；服务器会产生普通静态资源／公开API访问日志。未修改 `backend/data/`、`backend/server.linux`，未提交、推送、切换分支或执行reset／checkout／clean。保留远端候选与旧dist供回滚。

本轮本地临时目录 `E:\apps\github.com\sbu2video\.tmp_authrelease_20261008_01` 已清理（148项文件、12,236,167字节），删除前核对项目内准确绝对路径及无reparse point；删除后 `Test-Path=False`，脱敏网络记录仍存在。删除的是本轮构建副本、临时部署脚本和测试附件，可重新生成；产品媒体、验收文档与截图保留，远端候选及旧前端也保留供恢复。之前临时目录未触碰。

一个下一步：审阅107新版登录／注册及手机效果；下一轮性能范围可用专用测试账号扩大到真实登录、创作目录、钱包和任务列表。
