# 客户端 · 广场

## 页面职责

浏览官方场景模板与预设提示词，一键将场景描述带入创作页。只读浏览 + 跳转，不在广场页提交任务。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/garden` |
| 页面组件 | `frontend/src/pages/client/GardenPage.vue` |
| 下游 | 套用场景 → `/app/create`（写入 `draftPrompt`） |

## 业务流程

```text
进入广场
  → GET scenes（分类 + 列表）
  → 可选：按分类筛选、关键词搜索（前端或后端）
  → 点击「带入创作」
  → PATCH/POST 写入用户 draftPrompt
  → 跳转创作页
```

## 页面区块

### 1. 筛选 `filters`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `category` | `string` | 全部 / 电影感 / 自然 / 商业 / 生活 / 叙事 / 艺术 |
| `keyword` | `string` | 搜索标题与描述 |

### 2. 场景卡片 `scenes[]`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 如 `scene-1` |
| `title` | `string` | 场景名 |
| `category` | `string` | 分类 |
| `description` | `string` | 镜头结构 / 提示词说明 |
| `coverColor` | `string` | 卡片背景色（展示用，可选 hex） |
| `accentColor` | `string` | 强调色（展示用） |

套用逻辑：`draftPrompt = "{title}，{description}"`。

## 推荐接口

### 列表查询

```
GET /api/v1/client/scenes?category=电影感&keyword=雨夜
```

```json
{
  "items": [
    {
      "id": "scene-1",
      "title": "雨夜霓虹漫步",
      "category": "电影感",
      "description": "潮湿街巷、霓虹反射和稳定跟拍，适合人物氛围片。",
      "coverColor": "1e293b",
      "accentColor": "818cf8"
    }
  ],
  "categories": ["电影感", "自然", "商业", "生活", "叙事", "艺术"]
}
```

### 套用场景（可选独立接口）

```
POST /api/v1/client/drafts/prompt
{ "sceneId": "scene-1" }
```

或前端本地拼接后写入会话；后端可在用户 profile 存 `draftPrompt` 以跨设备同步。

```json
{ "draftPrompt": "雨夜霓虹漫步，潮湿街巷、霓虹反射和稳定跟拍，适合人物氛围片。" }
```

## 业务规则

1. 场景模板由**管理端**维护（产品方案中「场景模板」暂缓，首期可用静态配置）。
2. 广场不产生计费；仅影响创作页 `prompt` 初始值。
3. 搜索建议后端 `title` + `description` 模糊匹配；原型为前端 filter。

## 与当前 mock 对应

| 数据 | 来源 |
| --- | --- |
| `sceneTemplates` | `frontend/src/data/mock.ts` |
| 套用 | `app.draftPrompt = ...` + `router.push('/app/create')` |

管理端后续：`GET/POST /api/v1/admin/scene-templates`。
