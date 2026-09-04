# 客户端 · 创作

## 页面职责

创作页只提供文生视频、首尾帧生成视频、多图生成视频三种固定模式。创作者可以选择承运商和模型，或让系统智能推荐；供应商和采购成本始终不可见。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/create` |
| 页面组件 | `frontend/src/pages/client/CreatePage.vue` |
| 上游入口 | 首页概览、任务页新建、广场套用提示词 |
| 下游 | 创建成功后跳转 `/app/works` |

## 输入合同

| 模式代码 | 名称 | 输入 |
| --- | --- | --- |
| `text_to_video` | 文生视频 | `prompt` |
| `first_last_frame_to_video` | 首尾帧生成视频 | `prompt`、`sourceMeta.first`、`sourceMeta.last` |
| `multi_image_to_video` | 多图生成视频 | `prompt`、`sourceMeta.refs[]` |

图片引用均为 `{ "type": "upload" | "asset", "id": "..." }`。自动音频表示模型自动生成音频，不是参考音频上传。

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

## 报价

```http
POST /api/v1/client/tasks/quote
Content-Type: application/json

{
  "modeId": "mode_text_to_video",
  "modelId": "mdl_h3",
  "durationSeconds": 10,
  "resolution": "768P",
  "aspectRatio": "9:16",
  "generateAudio": false
}
```

```json
{
  "quoteId": "qt_a1b2c3d4",
  "expiresAt": "2026-09-04T08:05:00Z",
  "selection": {
    "carrier": { "id": "car_minimax", "name": "MINMAX" },
    "model": { "id": "mdl_h3", "name": "H3" }
  },
  "price": { "currency": "CNY", "pricePerSecond": 0.5, "durationSeconds": 10, "total": 5.0 }
}
```

用户选择明确模型时，服务端只会在该模型的可用供应商供给中选路；只有智能推荐才按模式模型策略选模型。报价有效期为 5 分钟。

## 创建任务

```http
POST /api/v1/client/tasks
Content-Type: application/json

{
  "quoteId": "qt_a1b2c3d4",
  "title": "雨夜街道",
  "prompt": "镜头缓慢向前推进……",
  "sourceMeta": {
    "first": { "type": "asset", "id": "ast_1" },
    "last": { "type": "upload", "id": "upl_2" }
  }
}
```

服务端在同一事务中校验报价未过期且供给/价格规则仍可用，锁定钱包，创建不可变任务快照，扣款写流水，并将报价标记为已使用。不得提交 `workflowId`、供应商、路由、成本、价格快照、时长或分辨率。

| 状态码 | 代码 | 场景 |
| --- | --- | --- |
| `400` | `INVALID_TASK_INPUT` | 模式输入或素材归属不合法 |
| `402` | `INSUFFICIENT_BALANCE` | 余额不足 |
| `409` | `QUOTE_EXPIRED_OR_UNAVAILABLE` | 报价已使用、过期、调价或供给下线 |

上游任务失败后服务端写退款流水。用户重试需要重新配置并取得新的报价。
