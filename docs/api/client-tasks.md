# 客户端 · 任务

## 页面职责

查看全部生成任务队列：排队、生成中、已完成、失败。支持终止退款、重试、下载结果、保存到当前作品。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/works` |
| 页面组件 | `frontend/src/pages/client/WorksPage.vue` |
| 分页 | `pageSize = 10` |

## 业务流程

```text
进入任务页
  → GET tasks（status 筛选 + 分页）
  → 展示各状态数量 counts
  → 操作分支：
      排队中/生成中 → POST cancel（终止并退款）
      失败         → POST retry（重新排队，不重复扣费）
      已完成       → GET download-url | POST save-to-project
```

## 列表筛选

顶部分段控件按 **任务状态** 筛选（不是额外「筛选」按钮）：

| `status` 参数 | 展示标签 |
| --- | --- |
| （空） | 全部 |
| `排队中` | 排队中 |
| `生成中` | 生成中 |
| `已完成` | 已完成 |
| `失败` | 失败 |

每项附带数量 `counts[status]`，与侧栏任务角标（活跃数 = 排队中 + 生成中）口径一致。

## 任务对象 `Task`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | `tsk_*` |
| `title` | `string` | 任务标题 |
| `workflow` | `string` | 工作流名称 |
| `status` | `TaskStatus` | 排队中 \| 生成中 \| 已完成 \| 失败 |
| `progress` | `integer` | 0–100；仅生成中展示 |
| `duration` | `string` | 展示用，如 `15 秒` |
| `resolution` | `string` | 如 `768P` |
| `cost` | `number` | 预扣/实扣金额 |
| `time` | `string` | 展示用时间 |
| `thumbUrl` | `string` | 列表缩略图 |
| `savedToProject` | `boolean` | 是否已保存到作品（可选） |

```ts
type TaskStatus = '排队中' | '生成中' | '已完成' | '失败'
```

## 推荐接口

### 列表

```
GET /api/v1/client/tasks?status=生成中&page=1&pageSize=10
```

```json
{
  "items": [ { "id": "tsk_88921", "title": "雨夜街道霓虹漫步", "status": "生成中", "progress": 68, "...": "..." } ],
  "total": 4,
  "page": 1,
  "pageSize": 10,
  "counts": {
    "全部": 4,
    "排队中": 0,
    "生成中": 1,
    "已完成": 2,
    "失败": 1
  }
}
```

### 终止并退款

```
POST /api/v1/client/tasks/{id}/cancel
```

```json
{
  "task": { "id": "tsk_88921", "status": "失败" },
  "refundAmount": 4.5,
  "wallet": { "balance": 260.5 }
}
```

仅 `排队中` / `生成中` 可取消；全额退回预扣。

### 重试

```
POST /api/v1/client/tasks/{id}/retry
```

失败任务重新进入 `排队中`；**不重复扣费**（沿用原预扣或标记为免费重试）。

### 下载

```
GET /api/v1/client/tasks/{id}/download-url
```

返回短时签名 URL 或 `302` 重定向。

### 保存到作品

```
POST /api/v1/client/tasks/{id}/save-to-project
{ "projectId": "prj_default" }
```

默认保存到 `{project}/视频/正片镜头/`。响应含新建 `FsItem`。

```json
{
  "asset": {
    "id": "ast-7",
    "name": "雨夜街道霓虹漫步.mp4",
    "type": "video",
    "sourceTask": "tsk_88918"
  },
  "path": ["视频", "正片镜头"]
}
```

## 业务规则

1. 进度更新：生成中任务建议 WebSocket / SSE 推送 `progress`，或轮询 `GET /tasks/{id}`。
2. `savedToProject`：同一 `taskId` 在同一作品内只保存一次。
3. 详情页原型为 toast；后续可增 `GET /tasks/{id}` 详情抽屉。

## 与当前 mock 对应

| 操作 | store |
| --- | --- |
| 列表 | `tasks` |
| 终止退款 | `cancelTask(id)` |
| 重试 | `retryTask(id)` |
| 保存作品 | `saveWorkToAsset(id)` |
| 已保存判断 | `isWorkSaved(taskId)` |
