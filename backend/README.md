# Sub2Video Backend

Gin + PostgreSQL 客户端 API。从 Sub2API 提取认证/钱包模式，不含 API 网关。

## 栈

| 项 | 选型 |
| --- | --- |
| HTTP | Gin |
| DB | PostgreSQL 16 |
| Auth | JWT access + refresh（bcrypt） |
| 端口 | 默认 `127.0.0.1:12080`（避免与前端 12000 / 常见 8080 冲突） |

## 本地启动

```bash
cd backend
docker compose up -d
go mod tidy
go run ./cmd/server
```

环境变量见 `internal/config/config.go`。默认：

- DB: `postgres://sub2video:sub2video@127.0.0.1:15432/sub2video?sslmode=disable`
- API: `127.0.0.1:12080`
- 演示账号: `demo@sub2video.local` / `Demo1234!`（余额 256）

## 认证接口

```
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/auth/me          # Bearer access token
```

## 已挂载客户端接口

```
GET  /api/v1/client/overview
GET  /api/v1/client/wallet
POST /api/v1/client/wallet/redeem
```

其余创作/任务/资产接口按 `docs/api/` 后续补齐。

## 107 部署

1. `docker compose up -d` 在 107 上起 PG/Redis（端口仅绑 127.0.0.1）
2. `GOOS=linux GOARCH=amd64 go build -o sub2video-server ./cmd/server`
3. 监听 `127.0.0.1:12080`，Nginx `/api` 反代
4. systemd 单元见 `deploy/sub2video-backend.service`
