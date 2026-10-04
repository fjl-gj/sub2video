# 客户端 · 任务

## 页面职责

查看生成任务、交付和退款状态。支持取消尚未进入执行边界的任务、恢复原参数创作、下载成片和选择目标作品保存。

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
      canCancel=true → POST cancel（服务端再次检查取消边界）
      reviewRequired → 查看核对提示，关联任务联系支持，不重复提交
      失败/已完成    → 确认恢复原参数为草稿，重新估价后创建新任务
      已交付         → GET tasks/{id}?download=true | POST save-to-project
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
| `mode` | `string` | 创作模式展示名 |
| `status` | `TaskStatus` | 排队中 \| 生成中 \| 已完成 \| 失败 |
| `progress` | `integer` | 0–100；仅生成中展示 |
| `duration` | `string` | 展示用，如 `15 秒` |
| `resolution` | `string` | 如 `768P` |
| `cost` | `number` | 预扣/实扣金额 |
| `time` | `string` | 展示用时间 |
| `thumbUrl` | `string` | 列表缩略图 |
| `savedToProject` | `boolean` | 是否已保存到作品（可选） |
| `canCancel` | `boolean` | 服务端判断当前是否允许普通取消；缺字段时客户端不推断允许 |
| `reviewRequired` | `boolean` | 提交结果待核对；不要重复提交或承诺已经退款 |

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

只有 `canCancel=true` 时展示取消入口；已经进入提交边界、已受理或受理结果不确定的任务不能普通取消。服务端事务是最终判定，状态竞争导致拒绝时保留错误，不展示退款成功。成功取消返回真实退款金额和最新余额。

### 重新配置创作

失败任务不能复用原任务或旧执行快照。客户端可以在用户确认后把原始创作参数恢复为新草稿，但仍必须重新请求 `POST /api/v1/client/tasks/estimate`，再通过 `POST /api/v1/client/tasks` 创建独立任务并按提交时售价预扣。

失败任务必须重新提交完整参数；不存在复用旧任务、旧价格或旧执行配置的重试接口。原任务详情只返回安全的模式、承运商、模型和素材引用，供草稿恢复使用。

### 下载

```
GET /api/v1/client/tasks/{id}?download=true
```

普通详情的 `downloadUrl` 用于视频预览；显式下载请求返回签入 `Content-Disposition: attachment` 的独立地址，客户端使用浏览器原生下载。两者都先验证任务归属，仅在 `deliveryStatus = "ready"` 时可用。签名失败返回 503，不回退到打开预览链接或虚报下载成功。地址来自平台对象存储，绝不返回供应商临时结果地址。`pending_storage` 表示生成完成但对象存储尚未交付。

### 任务详情

客户端路由为 `/app/works/{id}`。详情接口返回状态、进度、时长、分辨率、画幅、自动音频、已预扣金额、失败原因、退款金额/时间和交付状态，并可返回用于恢复草稿的安全创作参数；不返回供应商、采购成本、上游任务、上游工作流、路由或执行快照。

详情返回 `canCancel`、`reviewRequired`。只有真实退款流水存在时才声明已经退款。已保存的成片可返回 `savedDestination: { projectId, folderPath: string[] }`，来自持久作品目录；作品移入回收站或成片被删除时不返回失效位置。客户端刷新后仍可通过此字段定位保存目录。

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
3. 成片仅在对象存储持久化交付完成后可下载或保存到作品；没有交付地址时页面展示等待状态，不伪造下载成功。
4. 新任务失败后必须重新提交完整创作参数；调价、供给变更和余额变化都会影响重新提交的结果。

## 当前实现映射

| 操作 | 前端状态 / 接口 |
| --- | --- |
| 列表 | `GET /client/tasks` |
| 任务详情 / 视频预览 | `GET /client/tasks/{id}` |
| 附件下载 | `GET /client/tasks/{id}?download=true` |
| 终止退款 | `POST /client/tasks/{id}/cancel` |
| 失败重新配置 | 跳转 `/app/create`，由报价和创建任务接口完成 |
| 保存作品 | `POST /client/tasks/{id}/save-to-project` |
| 已保存判断 | 列表和详情响应的 `savedToProject` |
# 当前提交合同

客户端提交完整创作参数，使用 `Idempotency-Key` 保证幂等；服务端在事务中完成最低成本路由、消费订单、钱包扣费和任务创建。历史 `POST /client/tasks/quote` 现在返回 `410 LEGACY_QUOTE_DISABLED`，旧表仅保留已部署历史数据，不属于当前客户端流程。

浏览器在发送请求前按账号持久保存原请求编号与完整参数。响应丢失或损坏时保留记录，刷新后通过“确认原提交结果”复用同一请求，不能因编辑参数生成第二笔购买。不能保存恢复记录时不发送任务请求。前端先完成素材准备并取得费用预估，再开放普通提交；这不构成锁价，服务端仍在提交时重新计算。

`POST /client/tasks/estimate` 只计算当前可用的客户端价格，不创建任务、订单或 quoteId，也不锁定供给；它会确认本次候选供给凭据和对象存储可用。正式提交时服务端再次路由，并在扣费前确认对象存储桶可用。三种创作模式都必须经过对象存储检查，因为最终成片统一从对象存储交付。

普通取消只适用于尚未进入供应商提交边界的任务；已经开始提交或受理结果不确定时不会自动退款。`SUBMIT_UNKNOWN` 由管理员依据供应商记录，通过带审计的核对接口恢复轮询或确认未受理后退款，参见 `backend/README.md` 的恢复步骤。
