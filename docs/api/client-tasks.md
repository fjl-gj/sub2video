# 客户端 · 任务

## 页面职责

查看全部生成任务队列：排队、生成中、已完成、失败。支持终止退款、重新配置创作、下载结果、保存到当前作品。

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
      失败         → 跳转创作页，取得新报价并提交新任务
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
| `workflow` | `string` | 创作模式展示名；字段名为兼容旧任务保留 |
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

### 重新配置创作

失败任务不能复用原任务或旧报价。客户端跳转 `/app/create`，用户重新选择参数、请求新的 `POST /api/v1/client/tasks/quote`，再通过 `POST /api/v1/client/tasks` 创建独立任务并按当时售价预扣。

历史兼容接口 `POST /api/v1/client/tasks/{id}/retry` 不适用于带 `quoteId` 的新任务；服务端会要求重新配置和报价。

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
4. 新任务失败后必须使用新的 `quoteId`；调价、供给变更和余额变化都会影响重新提交的结果。

## 与当前 mock 对应

| 操作 | store |
| --- | --- |
| 列表 | `tasks` |
| 终止退款 | `cancelTask(id)` |
| 失败重新配置 | 跳转 `/app/create`，由报价和创建任务接口完成 |
| 保存作品 | `saveWorkToAsset(id)` |
| 已保存判断 | `isWorkSaved(taskId)` |
