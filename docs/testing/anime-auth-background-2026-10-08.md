# 登录／注册动漫背景接入验收 · 2026-10-08

## 结果与范围

已将用户提供的 `D:\download\pictures\sub2video\login_demo.mp4` 制作为无音轨循环版及匹配封面，接入本地登录、注册页。保留既有整屏背景、品牌字体、暖金主操作、输入控件及认证行为；1024px以下将裁切焦点从50%调整到30%。官网继续使用原海岸视频与其独立授权信息。

本轮没有提交、推送、部署或重启107，没有后端改动、登录／注册提交、任务创建、扣费或业务资产写入。107只经本地代理读取公开设置。既有导演台、后端、布局及测试修改全部保留；不将整个Git差异误记为此次新增改动。

## 测试版本与环境

- 分支：`codex/commercial-studio-redesign`。
- HEAD：`0cafa8d612f4ad8dcebcff58d7d25d674ebd5945`，有此前大量未提交修改；本次证据针对加入新媒体后的当前工作区，不等同于该HEAD的纯提交构建。
- Windows；Node `v22.16.0`；pnpm `10.32.0`；Vite `6.4.3`；Playwright `1.62.1`。
- 本地实页：`http://127.0.0.1:4186/auth/login` 与 `/auth/register`。Vite代理只读取107现有入口的公开设置；此次未使用登录凭据。
- 自动化：本地4185独立测试服务，未匹配API代理到 `127.0.0.1:9`，各业务请求由确定性夹具拦截。桌面及手机宽度均使用Chromium；手机项目是触控与视口模拟，不代表真实手机Safari。

## 媒体产物与可复核摘要

| 文件 | 大小／格式 | SHA-256 |
| --- | --- | --- |
| 原始 `login_demo.mp4` | 5,464,548字节；1344×768；24fps；10.125秒 | `c4d1be91dc1abdf261af57cc92ddbcab93e0c381c12e9abb628c750841872d53` |
| `frontend/public/brand/auth/creator-studio-loop.mp4` | 2,064,807字节；H.264；1344×768；24fps；231帧；9.625秒 | `4ecc7aca13fdb3a6cd2079c94632250ff958138f0c0ffcaf788e8833765b5838` |
| `frontend/public/brand/auth/creator-studio-poster.jpg` | 218,311字节；1344×768；JPEG质量90 | `3ab4555bf085d34c758d698ce5947f9c8e2f2555f5d0292cbb193b7464abefd9` |

处理使用FFmpeg 7.1（imageio-ffmpeg 0.6.0提供的临时工具），H.264 `-preset medium -crf 22 -pix_fmt yuv420p -movflags +faststart -an`。项目／系统没有新增运行时依赖。

生成方法：保留原第12–230帧，将末尾231–242帧与开头0–11帧做12帧余弦融合；再旋转48帧，将起点设在原第60帧的睁眼姿态。最终播放边界连接原59→60连续帧，融合段位于约7.125–7.625秒。封面取最终MP4第一帧，避免播放时由另一构图跳入。

文件结构解析确认：1条视频轨、0条音轨，`moov`位于`mdat`之前。OpenCV完整解码231帧；构建目录内的视频和封面SHA与源码资产一致。原片在生成前后和最终复核时SHA均一致。

首尾灰度差由11.45降至1.68，仅为选切点辅助指标，不是视觉无缝验收依据。融合中间帧存在轻微面部、发丝及手部柔化／双影；没有宣称完全无缝，也未重新生成素材或强行改变人物造型。

## 当前播放与响应式行为

- 视频与封面均为全视口 `cover` 背景，覆盖导航及表单背后，没有左右空白或不透明表单侧栏。
- 桌面保留58%水平焦点，右侧表单与暗部叠加；1024px以下30%焦点保留人物面部。
- 减少动态效果、省流或小屏首次进入时只显示封面，点击播放才请求视频；已在桌面播放的会话缩到手机可以继续播放，不属于手机首次加载视频。
- `muted`、`loop`、`playsinline`和可访问的播放／暂停按钮保留。标签页隐藏时暂停；自动播放遭浏览器拒绝时可手动播放。
- 视频失败移除播放器并显示封面状态；封面也失败时仍能使用表单。失败回退属于前端夹具故障注入证据。
- 登录／注册切换保持品牌媒体及内部回跳目的地；新媒体不显示旧海岸作者／CC链接。新来源没有继承旧素材许可。

## 实际内置浏览器检查

使用当前本地前端与真实107公开设置，未提交登录或注册表单。107当前开放注册、关闭邮箱验证和密码找回，因此实际截图可能少于开启全部开关的视觉测试表单。

| 实际视口 | 页面与结果 |
| --- | --- |
| 1440×900 | 登录：背景全幅，右侧400px表单，人物面部未被品牌文案覆盖；手动播放真实MP4成功 |
| 1920×1080 | 注册：全1920px背景，420px表单；无水平溢出；视频播放后 `duration=9.625`、`readyState=4`、`muted=true`、`loop=true` |
| 1024×768 | 注册：可用宽度1009px（原生纵向滚动条占15px），360px表单，文档宽度1009px；播放中视频保持铺满，表单可自然滚动 |
| 768×1024 | 注册：刷新后只有封面，400px表单，文档／可用宽度均768px |
| 390×844 | 登录和注册：刷新后无video元素；表单约358px，文档／可用宽度390px；人物焦点30% |
| 320×480 | 注册：可用宽度305px，文档宽度305px；滚动到最终操作、返回登录、播放及法律链接，未被背景截获或遮挡 |

实页视频时间持续推进，观测到9.17秒后的4.75秒且仍播放；自动化另通过靠近末尾的真实解码视频验证原生循环回绕和继续推进。静态截图不作为连续动作观感或供应商交付证据。

已保存当前实际页面截图：

- [桌面登录](anime-auth-login-desktop-2026-10-08.jpg)
- [手机登录](anime-auth-login-mobile-2026-10-08.jpg)
- [宽屏注册](anime-auth-register-desktop-2026-10-08.jpg)
- [手机注册](anime-auth-register-mobile-2026-10-08.jpg)
- [320px短屏注册完整页面](anime-auth-register-short-2026-10-08.jpg)

## 检查命令与结果

| 验证 | 命令／证据 | 结果与边界 |
| --- | --- | --- |
| 静态检查 | `pnpm lint:check` | 通过；前端源码规则 |
| 类型检查 | `pnpm typecheck` | 通过；Vue/TypeScript |
| 生产构建 | `pnpm build --outDir ../.tmp_authvideo_20261008_01/dist` | 通过；2472模块；媒体产物SHA匹配，未部署 |
| 针对性媒体／布局测试 | `node node_modules/@playwright/test/cli.js test -c ../.tmp_authvideo_20261008_01/playwright.auth.config.ts auth-forms.contract.spec.ts full-background.contract.spec.ts public-home.contract.spec.ts` | 134 passed、8 skipped、0失败；真实本地媒体与Mock业务响应 |
| 视觉更新 | 同CLI与配置，`auth.visual.spec.ts --update-snapshots` | 仅4张认证基线更新；逐张人工审图，保留原容差 |
| 完整前端回归 | `node node_modules/@playwright/test/cli.js test -c ../.tmp_authvideo_20261008_01/playwright.auth.config.ts` | 436 passed、36 skipped、0 unexpected、0 flaky；3.1分钟，包含更新后基线的正常比较 |
| 差异检查 | `git diff --check`，未跟踪文本另独立检查 | 通过；仅LF/CRLF提示，不是空白错误 |

完整回归与 `pnpm test:visual` 使用同一基础Playwright配置及全部用例，临时覆盖仅用于独立测试结果目录和绝对webServer cwd。使用Node直接调用CLI，避免本机pnpm exec参数转义曾造成的 `unknown command '^test^'`。浏览器／Vite子进程在默认沙箱遇到 `spawn EPERM` 后，使用经自动审批允许的本地进程权限执行；没有绕过断言或增加重试。

36个跳过均为避免重复运行同一尺寸矩阵或最小宽度场景：auth-forms 1、journey 1、studio 1、public-home 7、responsive-layout 12、ui-standards 14。对应场景已在另一project执行。没有基础设施缺失导致的跳过。针对性运行的8个跳过也属于同类设计。

## 本轮修改文件

- 新增静音MP4及JPG封面两项产品素材。
- `frontend/src/features/auth/AuthShowcase.vue`：媒体地址、视频poster、可访问描述和品牌媒体说明；保留既有播放策略及故障处理。
- `frontend/src/features/auth/auth.css`：小屏背景焦点30%；原有整屏结构保留。
- `frontend/tests/e2e/auth-forms.contract.spec.ts`：新媒体／署名、真实原生循环、切换及故障合同。
- `frontend/tests/e2e/full-background.contract.spec.ts`：区分官网／认证媒体，故障拦截覆盖WebM与MP4。
- `frontend/tests/e2e/auth.visual.spec.ts-snapshots/`：登录／注册桌面与手机4张有意更新，未变更其他基线。
- `frontend/public/brand/auth/ATTRIBUTION.md`、`docs/ui/auth-entrance.md`、`docs/ui/design-system.md`：更新当前媒体来源及构图约定，保留历史说明。
- 本记录及上述5张实际页面截图。

## 数据影响、清理与剩余限制

本轮临时目录 `E:\apps\github.com\sbu2video\.tmp_authvideo_20261008_01` 已清理：164项文件、95,450,063字节，包含下载的临时编码工具、脚本、构建和测试中间产物。删除前确认准确绝对路径处于项目内且不存在reparse point；删除后 `Test-Path=False`，产品视频与封面仍存在。证据已整理到本记录，不保留该目录原始日志；编码／构建／测试产物可按上述方法重新生成。新媒体、实际截图及4张视觉基线是有意交付，不属于应清理的临时文件。以前临时目录未触碰。

没有数据库迁移、钱包或用户资产影响；原视频及原图片未修改。没有提交、推送、107部署。保留本地4186预览供用户查看。

未验证：真机iOS/Safari、手机软键盘、安全区的实际设备表现、真实登录／注册与邮件发送。没有重新做供应商、任务交付或支付验收；这些不由背景素材回归替代。素材短融合的轻微柔化仍需用户审美评审；来源和处理记录不等同于独立版权核验。

下一步：用户在本地连续预览登录／注册效果，确认该片的过渡与移动裁切，再确定是否发布到107。
