
# 危险废物转运合规核验

产废单位、承运资质、转运清单和合规核验平台。项目采用前后端分离和明确的领域分层，重点保证状态迁移、RBAC、审计日志、请求追踪与限流在各层保持一致。

## Docker Compose 快速启动

```bash
cp .env.example .env
docker compose up -d --build
docker compose ps
```

- Web 工作台：http://127.0.0.1:18518
- 后端健康检查：http://127.0.0.1:19518/healthz
- 后端 API：http://127.0.0.1:19518/api
- 本地演示管理员：`admin` / `Admin123!`。演示密码仅用于本地验收，生产环境必须删除种子账号并替换密钥。

| 演示账号 | 密码 | 角色 |
|---|---|---|
| `viewer` | `Admin123!` | 只读查看 |
| `operator` | `Admin123!` | 建档、编辑、推进联单 |
| `reviewer` | `Admin123!` | 复核许可、作出核验决定、查看审计 |
| `admin` | `Admin123!` | 全部权限与受控删除 |

停止并清理本项目容器与数据卷：

```bash
docker compose down -v --remove-orphans
```

## 主要功能

| 业务模块 | 后端实体 | API 前缀 | 状态流 |
|---|---|---|---|
| 产废单位 | `WasteGenerator` | `/api/generators` | 许可编号、有效期、废物类别与证据 |
| 承运资质 | `CarrierProfile` | `/api/carriers` | 许可证、有效期、有效车辆与证据 |
| 转运清单 | `TransferManifest` | `/api/manifests` | 产废单位、承运方、废物代码、重量与去向 |
| 合规核验 | `ComplianceCheck` | `/api/checks` | 关联联单、核验清单、证据与决定依据 |

- JWT 登录和 viewer/operator/reviewer/admin 四级 RBAC，后端 middleware、前端守卫、导航与按钮同步生效。
- 联单提交和发运前会重新核验产废许可为 `active`、承运资质为 `verified`，且双方证照仍在有效期内。
- 联单只允许 `draft → submitted → in_transit → received`，`submitted/in_transit` 可转 `rejected`；核验决定不可回退，失败仅可升级复核。
- 已提交联单和已决定核验不可编辑或删除；写入使用乐观锁。
- 建档、许可/证据更新、状态变化和删除与审计日志在同一数据库事务中提交，审计保留 actor 与 request ID。
- 请求 ID、结构化日志、全局错误映射和 Redis 分布式限流。
- 提供脱敏运行配置、当前会话、审计汇总和单实体审计历史接口。
- 业务工作台支持查询、分页、新建、分支状态推进、许可到期摘要、风险标识及操作审计查看。

## 权限矩阵

| 操作 | viewer | operator | reviewer | admin |
|---|:---:|:---:|:---:|:---:|
| 查看四类业务数据 | ✓ | ✓ | ✓ | ✓ |
| 新建和编辑待处理数据 |  | ✓ | ✓ | ✓ |
| 推进转运联单 |  | ✓ | ✓ | ✓ |
| 复核许可与核验决定 |  |  | ✓ | ✓ |
| 查看审计 |  |  | ✓ | ✓ |
| 受控软删除 |  |  |  | ✓ |

## 技术栈

| 层次 | 技术 |
|---|---|
| 前端 | Angular 17 + TypeScript + Vite + Angular Material |
| 后端 | Go 1.22 + Gin + GORM |
| 数据 | PostgreSQL + Redis、MinIO |
| 部署 | Docker Compose + Nginx |

## 本地开发

后端可使用 SQLite 开发模式，不需要先启动数据库：

```bash
cd backend
go mod download
DATABASE_DRIVER=sqlite DATABASE_DSN=local.db REDIS_ADDR='' \
JWT_SECRET=local-development-secret PORT=8080 go run ./cmd/server
```

前端开发服务器：

```bash
cd frontend
npm install
npm run dev
```

质量检查：

```bash
cd backend && go test -race ./... && go vet ./... && go build ./...
cd ../frontend && npm run typecheck && npm run build
cd .. && docker compose config --quiet
```

也可以从项目根目录执行 `./scripts/validate.sh`。脚本会从空命名卷构建并启动全部服务，验证 RBAC、关联资质阻断、状态跳级阻断、乐观锁、核验决定和 request ID 审计，结束时关闭容器并删除本项目卷。设置 `KEEP_RUNNING=1` 可为内置 Browser 验收保留服务。

## 目录结构

```text
.
├── backend/
│   ├── cmd/server/                 # 服务入口与优雅退出
│   └── internal/
│       ├── config/                 # 环境配置
│       ├── constants/              # 状态枚举与迁移图
│       ├── database/               # 连接、迁移与演示数据
│       ├── dto/                    # 输入契约
│       ├── handler/                # HTTP 接口
│       ├── middleware/             # JWT、追踪、限流
│       ├── model/                  # GORM 实体
│       ├── repository/             # 持久化边界
│       ├── router/                 # 路由装配
│       ├── service/                # 业务规则与审计
│       └── util/                   # 统一 HTTP 响应
├── frontend/src/
│   ├── api/                        # 按实体拆分的 API
│   ├── components/common/          # 共享业务组件
│   ├── hooks/                      # 认证与分页 hooks
│   ├── pages/                      # 五个路由页面
│   ├── router/                     # 路由配置
│   ├── stores/                     # 按实体拆分的状态仓库
│   ├── types/                      # 共享类型与枚举
│   └── utils/                      # 格式化与状态工具
├── docker-compose.yml
├── runtime_smoke.json
└── VALIDATION.md
```

## 共享枚举位置

| 枚举 | 值 | 前后端出现位置 |
|---|---|---|
| `ManifestState` | `draft, submitted, in_transit, received, rejected` | `backend/internal/constants/status.go`、`frontend/src/types/status.ts` |
| `CheckState` | `pending, pass, fail, escalated` | `backend/internal/constants/status.go`、`frontend/src/types/status.ts` |

每个实体自己的完整迁移图同样位于 `backend/internal/constants/status.go`；页面使用的状态列表位于 `frontend/src/types/status.ts`。修改状态时必须同步两处并更新对应服务测试。

## 环境变量

| 变量 | 说明 |
|---|---|
| `COMPOSE_PROJECT_NAME` | 固定英文 Compose 项目名，支持中文父目录 |
| `DB_NAME/DB_USER/DB_PASSWORD` | 数据库名称与业务账号 |
| `DB_ROOT_PASSWORD` | MySQL 管理员密码（PostgreSQL 项目保留统一模板字段） |
| `JWT_SECRET` | JWT 签名密钥，生产环境必须替换 |
| `FRONTEND_PORT/BACKEND_PORT/DB_PORT` | 宿主机端口映射 |
| `REDIS_PORT` | Redis 宿主机端口 |
| `MINIO_*` | 证据对象存储配置（启用 MinIO 的项目） |

## API 使用示例

```bash
token=$(curl -sS -X POST http://127.0.0.1:19518/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin123!"}' | jq -r '.data.token')

curl -sS http://127.0.0.1:19518/api/overview \
  -H "Authorization: Bearer $token"
```

## License

MIT
