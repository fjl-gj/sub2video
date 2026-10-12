# 客户端 · 创作

## 页面职责

创作页只提供文生视频、首尾帧生成视频、多图生成视频三种固定模式。创作者可以选择承运商和模型，或让系统智能推荐；供应商和采购成本始终不可见。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/create` |
| 页面组件 | `frontend/src/pages/client/CreatePage.vue` |
| 上游入口 | 首页概览、任务页新建、广场套用提示词 |
| 下游 | 创建成功后跳转 `/app/works/:id` |

## 输入合同

| 模式代码 | 名称 | 输入 |
| --- | --- | --- |
| `text_to_video` | 文生视频 | `prompt` |
| `first_last_frame_to_video` | 首尾帧生成视频 | `prompt`、`sourceMeta.first`、`sourceMeta.last` |
| `multi_image_to_video` | 多图生成视频 | `prompt`、`sourceMeta.refs[]` |

图片引用均为 `{ "type": "upload" | "asset", "id": "..." }`。音频设置由客户端选择：不使用音频、自动生成音频或上传参考音频。后两种都提交 `generateAudio: true` 并按音频售价预估；参考音频额外写入 `sourceMeta.audio`，仅支持上传引用。

## 上传参考音频

```http
POST /api/v1/client/uploads/reference-audio
Content-Type: multipart/form-data
```

表单字段为 `file`，支持 MP3、WAV、FLAC、M4A、MP4，最大 50MB。上传只在用户确认提交参考音频任务时发生；响应的 `id` 写入任务素材快照。客户端响应始终不返回供应商、工作流编码或上游地址。

## 获取目录

```http
GET /api/v1/client/create/catalog
```

```json
{
  "modes": [{ "id": "mode_text_to_video", "code": "text_to_video", "name": "文生视频", "enabled": true }],
  "carriers": [{ "id": "car_minimax", "name": "MINMAX" }],
  "models": [{ "id": "mdl_h3", "carrierId": "car_minimax", "name": "H3", "modes": ["text_to_video"] }]
}
```

响应不含供应商、成本、上游工作流或路由。

## 获取参数能力

```http
GET /api/v1/client/create/options?modeId=mode_text_to_video&modelId=mdl_h3
```

`modelId` 可省略，代表智能推荐。响应只包含当前可报价能力的交集：

```json
{
  "durations": [1, 2, 3, 4, 5],
  "resolutions": ["768P"],
  "aspectRatios": ["16:9", "9:16", "1:1"],
  "supportsGeneratedAudio": false,
  "minImages": 0,
  "maxImages": 0,
  "defaultDuration": 5,
  "defaultResolution": "768P"
}
```

页面不得把 10 秒、1080P 或音频开关硬编码为全模型可用。

## 预估

```http
POST /api/v1/client/tasks/estimate
Content-Type: application/json

{
  "modeId": "mode_text_to_video",
  "carrierId": "car_minimax",
  "modelId": "mdl_h3",
  "durationSec": 10,
  "resolution": "768P",
  "aspectRatio": "9:16",
  "generateAudio": false,
  "sourceMeta": {
    "refs": [{ "type": "asset", "id": "ast_1" }]
  }
}
```

```json
{
  "available": true,
  "selection": {
    "carrier": { "id": "car_minimax", "name": "MINMAX" },
    "model": { "id": "mdl_h3", "name": "H3" }
  },
  "billing": {
    "basis": "按秒预估",
    "currency": "CNY",
    "pricePerSecond": 0.03,
    "durationSec": 10,
    "estimatedTotal": 0.30
  }
}
```

预估只计算当前可用能力和平台售价，不创建报价、订单或任务，不扣费，也不锁定执行配置。`sourceMeta` 只用于计算视觉素材数量，服务端不会在预估阶段上传或读取素材内容；提交阶段仍会重新校验素材归属。响应不包含供应商、执行类型、工作流编码、成本、Token 或上游地址。

## 创建任务

```http
POST /api/v1/client/tasks
Content-Type: application/json
Idempotency-Key: create_20260911_001

{
  "modeId": "mode_first_last",
  "carrierId": "car_minimax",
  "modelId": "mdl_h3",
  "durationSec": 10,
  "resolution": "768P",
  "aspectRatio": "9:16",
  "generateAudio": false,
  "title": "雨夜街道",
  "prompt": "镜头缓慢向前推进……",
  "sourceMeta": {
    "first": { "type": "asset", "id": "ast_1" },
    "last": { "type": "upload", "id": "upl_2" }
  }
}
```

提交时服务端重新选择可用执行配置并计算最终价格；它不会使用或校验预估结果。在同一事务中校验素材归属、锁定钱包、创建消费订单、写入扣费流水及不可变任务快照。同一用户重复使用同一幂等键时，返回原订单与原任务，不会重复扣费。参考音频按如下方式随任务保存：

```json
{
  "sourceMeta": {
    "audio": { "type": "upload", "id": "upl_audio_1" }
  }
}
```

不得提交 `workflowId`、供应商、路由、成本、价格快照或上游参数。

| 状态码 | 代码 | 场景 |
| --- | --- | --- |
| `400` | `PROMPT_REQUIRED` | 缺少画面描述 |
| `402` | `INSUFFICIENT_BALANCE` | 余额不足 |
| `400` | `UNAVAILABLE_COMBINATION` | 模式输入、素材归属或执行能力不满足要求 |
| `503` | `STORAGE_UNAVAILABLE` | 对象存储配置或访问不可用 |
| `503` | `SUPPLY_UNAVAILABLE` | 执行凭据或执行配置暂不可用 |
| `410` | `LEGACY_QUOTE_DISABLED` | 调用了已废弃的 `/client/tasks/quote` |

上游任务失败后服务端写退款流水。用户重试需要重新配置并提交新的请求。

## 导演台镜头上下文

从作品导演台明确应用镜头后，创建任务可额外提交：

```json
{
  "directorContext": {
    "projectId": "prj_example",
    "shotId": "shot_001",
    "expectedRevision": 2
  }
}
```

估价接口无需该字段，仍不产生持久记录。提交时先查询用户与幂等键对应的原订单；已有订单直接返回，不检查当前镜头或供给。首次请求才锁定作品并校验镜头修订号，和任务、订单、扣费一起写入候选关联与实际创作输入。版本冲突或镜头不存在会整体回滚，不能先创建任务再补关联。

前端将上下文与原完整参数一同冻结到待确认请求；网络结果不确定时只重放原请求。`409 DIRECTOR_BOARD_CONFLICT`、`409 DIRECTOR_SHOT_MISSING`、`404 NOT_FOUND` 是镜头/作品的明确拒绝，需要重新读取并由用户确认后再发起新请求。`404 NOT_FOUND` 的恢复判定仅限带导演上下文的请求，不能泛化为所有代理 404。

导演台首版仅支持无音频或能力允许的自动音频，不接入参考音频。普通创作的既有音频合同保持不变。镜头板保存与候选接口详见 [导演台数据合同](../ui/director-workspace-2026-10-05.md)。
