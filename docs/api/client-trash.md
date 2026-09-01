# 客户端 · 回收站

## 页面职责

存放从资产页移出的作品。支持**恢复**到资产列表，或**彻底删除**（不可恢复）。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/trash` |
| 页面组件 | `frontend/src/pages/client/TrashPage.vue` |
| 侧栏角标 | `trashedProjects.length` |

## 业务流程

```text
资产页「移入回收站」
  → POST /projects/{id}/trash
  → 作品 trashed=true，侧栏回收站 +1

回收站页
  → GET /projects?trashed=true
  → 恢复：POST /projects/{id}/restore
  → 彻底删除：DELETE /projects/{id}（需确认弹窗）
```

## 列表项 `TrashedProject`

继承 `WorkProject` 回收态字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 作品 ID |
| `name` | `string` | 作品名 |
| `desc` | `string` | 描述 |
| `trashedAt` | `string` | 移入时间 |
| `fileCount` | `integer` | 作品内文件数（含目录树） |

## 推荐接口

### 列表

```
GET /api/v1/client/projects?trashed=true
```

```json
{
  "items": [
    {
      "id": "prj_2",
      "name": "春日短片",
      "desc": "按标准工程模板创建",
      "trashedAt": "2026-09-01T10:00:00+08:00",
      "fileCount": 3
    }
  ]
}
```

### 恢复

```
POST /api/v1/client/projects/{id}/restore
```

`trashed` → `false`；若用户无当前作品，恢复项设为 `currentProjectId`。

### 彻底删除

```
DELETE /api/v1/client/projects/{id}?permanent=true
```

仅当 `trashed === true` 时可调用；删除 OSS 上全部对象。

## 业务规则

1. 资产页**不能**永久删除作品，只能移入回收站。
2. 回收站内作品仍占用存储配额（待产品确认；原型仍计入 `fileCount`）。
3. `prj_default` 是否允许进回收站：原型允许，生产可限制。
4. 彻底删除需二次确认；无批量操作（首期）。

## 与当前 mock 对应

| 操作 | store |
| --- | --- |
| 列表 | `trashedProjects` |
| 恢复 | `restoreProject(id)` |
| 彻底删除 | `purgeProject(id)` |
| 移入 | `trashProject(id)`（资产页触发） |
