# 客户端 · 创作

## 页面职责

选择工作流、上传或选取首帧素材、填写提示词与生成参数，确认报价后提交视频生成任务。提交成功跳转任务页。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/create` |
| 页面组件 | `frontend/src/pages/client/CreatePage.vue` |
| 上游入口 | 首页概览、任务页「新建创作」、广场「套用场景」 |
| 下游 | 提交后 → `/app/works` |

## 业务流程

```text
进入创作页
  → GET workflows（可用工作流 + 参数 schema）
  → 可选：从广场带入 draftPrompt
  → 选择工作流（首期仅「图生视频」可用）
  → 上传首帧 / 从当前作品资产选图
  → 填写 prompt、标题、比例、时长
  → POST quote（试算费用）
  → POST tasks（预扣余额、创建任务）
  → 跳转任务列表
```

## 页面区块与数据

### 1. 工作流选择 `workflows[]`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 如 `wf-h3-image` |
| `name` | `string` | 图生视频 / 文生视频 / 多图生视频 |
| `description` | `string` | 简短说明 |
| `enabled` | `boolean` | 是否可选；未开放为 `false` |
| `badge` | `string` | 可用 / 即将开放 |

首期仅 `wf-h3-image`（H3 图生视频）`enabled: true`。

### 2. 输入素材

| 方式 | 请求 | 说明 |
| --- | --- | --- |
| 本地上传 | `POST /api/v1/client/uploads` → `uploadId` | JPG/PNG，≤10MB；原型为模拟 |
| 从资产选择 | `GET /api/v1/client/projects/{projectId}/assets?type=image` | 当前作品下图片列表 |

| 资产项字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | `ast-*` |
| `name` | `string` | 文件名 |
| `folder` | `string` | 所在目录名（扁平展示用） |

提交任务时二选一：`uploadId` 或 `assetId`。

### 3. 画面描述 `prompt`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `prompt` | `string` | 主体动作、镜头、环境 |
| `suggestedTags` | `string[]` | 可选快捷词，如电影感、缓慢推进 |

广场套用场景时写入 `draftPrompt`，进入创作页预填。

### 4. 生成设置

| 字段 | 类型 | 枚举/约束 | 说明 |
| --- | --- | --- | --- |
| `title` | `string` | 可选 | 作品标题；空则用 prompt 截断 |
| `aspectRatio` | `string` | `16:9` \| `9:16` \| `1:1` | 画面比例 |
| `durationSeconds` | `integer` | 6 \| 10 \| 15 | 自定义暂未开放 |
| `resolution` | `string` | 首期固定 `768P` | 由工作流决定 |

### 5. 费用试算 `quote`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `estimatedCost` | `number` | 预计扣费（元） |
| `billingNote` | `string` | 如「按 768P 输出秒数计费」 |
| `pricePerSecond` | `number` | 参考单价 |
| `walletBalance` | `number` | 页头展示用 |

## 推荐接口

### 获取创作页初始化数据

```
GET /api/v1/client/create/form
```

```json
{
  "wallet": { "balance": 256.0 },
  "workflows": [
    {
      "id": "wf-h3-image",
      "name": "图生视频",
      "description": "一张图片 + 文字描述",
      "enabled": true
    }
  ],
  "draftPrompt": "",
  "currentProjectId": "prj_default",
  "parameters": {
    "aspectRatios": ["16:9", "9:16", "1:1"],
    "durations": [6, 10, 15],
    "defaultDuration": 15,
    "resolution": "768P"
  },
  "suggestedTags": ["电影感", "缓慢推进", "自然光", "细节丰富"]
}
```

### 试算费用

```
POST /api/v1/client/tasks/quote
Content-Type: application/json

{
  "workflowId": "wf-h3-image",
  "durationSeconds": 15,
  "resolution": "768P"
}
```

```json
{
  "estimatedCost": 7.5,
  "priceSnapshot": {
    "ruleId": "pr-1",
    "version": "v1.2",
    "pricePerSecond": 0.5
  }
}
```

### 提交任务

```
POST /api/v1/client/tasks
Content-Type: application/json

{
  "workflowId": "wf-h3-image",
  "title": "雨夜街道",
  "prompt": "镜头缓慢向前推进……",
  "aspectRatio": "16:9",
  "durationSeconds": 15,
  "resolution": "768P",
  "source": { "type": "asset", "assetId": "ast-2" },
  "priceSnapshot": { "ruleId": "pr-1", "version": "v1.2", "estimatedCost": 7.5 }
}
```

**Response `201`**

```json
{
  "task": {
    "id": "tsk_89004",
    "title": "雨夜街道",
    "status": "排队中",
    "cost": 7.5,
    "workflow": "H3 图生视频"
  },
  "wallet": { "balance": 248.5 },
  "ledgerEntryId": "TX_90201"
}
```

**错误**

| 状态码 | 场景 |
| --- | --- |
| `402` | 余额不足 |
| `400` | 工作流未开放、缺少素材或参数非法 |
| `409` | 价格快照过期，需重新 quote |

## 业务规则

1. 提交时**预扣** `estimatedCost`；失败或用户终止后退款（见任务域）。
2. `priceSnapshot` 必须在提交时一并传入，防止调价期间争议。
3. 从资产选图仅限**当前作品**（`currentProjectId`）下的图片。
4. 任务默认关联当前作品；保存成片时使用同一 `projectId`。

## 与当前实现对应

| 行为 | 实现 |
| --- | --- |
| 提交任务 | `POST /client/tasks` + CreatePage `createTask` |
| 本地上传 | `POST /client/uploads`，提交时带 `source.uploadId` |
| 余额展示 | auth/me + CreatePage 同步 `app.balance` / `auth.setBalance` |
| 广场预填 | `draftPrompt` |
| 资产列表 | `assets`（当前作品扁平化） |
| 工作流列表 | 页面内硬编码 `workflows` 数组 |
| 费用试算 | 页面内 `duration === '10 秒' ? 5 : 7.5` |

类型：`Task`（`frontend/src/data/mock.ts`）。
