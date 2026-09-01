# 客户端 · 资产

## 页面职责

以**作品（Project）**为单位的工程文件空间。每个作品套用标准目录模板，支持浏览、上传、删除文件；作品删除仅移入回收站。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/assets` |
| 页面组件 | `frontend/src/pages/client/AssetsPage.vue` |
| 类型定义 | `frontend/src/data/projects.ts` |

## 标准工程模板

每个作品创建时自动生成，**系统目录不可删除、不可改名**：

```text
{作品名}/
├── 资产/          （系统根目录）
│   ├── 人物/
│   ├── 场景/
│   ├── 道具/
│   └── 素材/
└── 视频/          （系统根目录）
    └── 正片镜头/   （默认成片目录，可扩展子文件夹）
```

常量：`SYSTEM_ROOTS = ['资产', '视频']`，`ASSET_BUCKETS = ['人物','场景','道具','素材']`，`DEFAULT_VIDEO_ALBUM = '正片镜头'`。

## 页面状态机

```text
[作品列表] ──进入作品──► [目录浏览]
     │                        │
     │ 新建作品               │ 面包屑导航 / 打开子文件夹
     │ 移入回收站             │ 上传素材 / 删除文件
     └────────────────────────┘
```

| 视图 | `inside` | 数据 |
| --- | --- | --- |
| 作品列表 | `false` | `activeProjects[]` |
| 目录浏览 | `true` | `currentProject` + `path[]` |

## 核心类型

### `WorkProject`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | `prj_*`；默认 `prj_default` |
| `name` | `string` | 作品名 |
| `desc` | `string` | 描述 |
| `createdAt` | `string` | 创建时间 |
| `trashed` | `boolean` | 是否在回收站 |
| `folders` | `Record<string, FsFolder>` | 根级目录树 |

### `FsFolder`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `isSystem` | `boolean` | 系统固定目录 |
| `desc` | `string` | 目录说明 |
| `folders` | `Record<string, FsFolder>` | 子目录 |
| `items` | `FsItem[]` | 当前层文件 |

### `FsItem`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | `ast-*` |
| `name` | `string` | 文件名 |
| `type` | `image` \| `video` \| `audio` | 文件类型 |
| `size` | `string` | 展示用大小 |
| `time` | `string` | 更新时间展示 |
| `meta` | `string` | 可选，如分辨率时长 |
| `sourceTask` | `string` | 来源任务 ID（成片） |

## 页面区块

### 1. 存储概览（列表与浏览共用）

| 字段 | 说明 |
| --- | --- |
| `storage.usedGb` / `quotaGb` | 进度条 |
| `storage.fileCount` | 活跃作品文件总数 |
| `storage.projectCount` | 活跃作品数 |

### 2. 作品列表

| 操作 | 接口 |
| --- | --- |
| 列表 | `GET /projects?trashed=false` |
| 新建 | `POST /projects` body: `{ name }` |
| 进入 | 客户端记 `currentProjectId` |
| 移入回收站 | `POST /projects/{id}/trash` |

卡片展示：`name`, `desc`, `createdAt`, `fileCount`, `isDefault`。

### 3. 目录浏览

| 操作 | 接口 |
| --- | --- |
| 列出目录 | `GET /projects/{id}/nodes?path=资产/人物` |
| 上传 | `POST /projects/{id}/uploads?path=资产/素材` |
| 删除文件 | `DELETE /projects/{id}/files/{fileId}` |

`path` 为 `/` 分隔的面包屑路径。响应：

```json
{
  "path": ["资产", "人物"],
  "folders": [
    { "name": "子文件夹", "isSystem": false, "desc": "...", "itemCount": 3 }
  ],
  "items": [
    { "id": "ast-1", "name": "人物侧脸素材.png", "type": "image", "size": "3.1 MB", "time": "昨天 19:22" }
  ]
}
```

## 推荐接口汇总

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/client/storage/usage` | 配额与用量 |
| GET | `/api/v1/client/projects` | 作品列表 `?trashed=false` |
| POST | `/api/v1/client/projects` | 按模板创建 |
| GET | `/api/v1/client/projects/{id}/tree` | 可选：整树；大作品用 nodes 分页 |
| GET | `/api/v1/client/projects/{id}/nodes` | 单层目录 `?path=` |
| POST | `/api/v1/client/projects/{id}/uploads` | multipart，`path` query |
| DELETE | `/api/v1/client/projects/{id}/files/{fileId}` | 删除非系统目录内文件 |
| POST | `/api/v1/client/projects/{id}/trash` | 移入回收站 |
| PUT | `/api/v1/client/session/current-project` | `{ projectId }` 切换当前作品 |

## 业务规则

1. **default作品**（`prj_default`）可进入、可上传，是否允许移入回收站由产品决定（原型允许）。
2. 系统目录 `isSystem: true` 不可删除；`资产` 下四个桶不可删。
3. 任务「保存到作品」写入 `视频/正片镜头/`，见 [任务](./client-tasks.md)。
4. 创作页「从资产选择」使用 `GET .../assets?type=image`（当前作品图片扁平列表）。
5. 回收站流程见 [回收站](./client-trash.md)。

## 与当前 mock 对应

| 行为 | store / 模块 |
| --- | --- |
| 作品列表 | `activeProjects` |
| 当前作品 | `currentProjectId`, `currentProject` |
| 创建 | `createProject(name)` |
| 移入回收站 | `trashProject(id)` |
| 上传 | `uploadAsset(projectId, path)` |
| 删除文件 | `removeAsset(id)` |
| 目录解析 | `resolveNode(project, path)` |
