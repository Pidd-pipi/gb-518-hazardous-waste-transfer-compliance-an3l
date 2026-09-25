请生成 `hazardous-waste-transfer-compliance`「危险废物转运合规核验」Go 全栈项目，面向环保企业管理产废单位、承运方、转运清单和合规核验决定。不要做电商、仓库库存、财务对账或工单客服。

## 项目主要需求

复杂度下限：核心实体不少于 3 个、核心页面不少于 4 个、横切关注点不少于 2 个、共享前端组件不少于 3 个、自定义 hooks/utils 不少于 2 个、后端中间件不少于 2 个。

### 核心实体

`WasteGenerator`（产废单位与许可）、`CarrierProfile`（承运资质）、`TransferManifest`（转运清单与状态）、`ComplianceCheck`（核验项与决定）贯穿数据库、Go model/service/handler 和前端。

### 核心页面

`/generators` 产废单位；`/carriers` 承运资质；`/manifests` 转运清单；`/checks` 合规核验；`/audit` 审计。`StatusBadge` 在清单和核验页共用，`LicensePanel` 在单位和承运页共用。

### 横切关注点

RBAC 联动角色、Go middleware、前端守卫和按钮；清单状态、资质文件和核验结果必须写操作审计与 request ID；全局错误处理和限流独立实现。

### 共享枚举/组件

同步 `ManifestState`（draft/submitted/in_transit/received/rejected）与 `CheckState`（pending/pass/fail/escalated）。共享 `StatusBadge`、`LicensePanel`、`ConfirmDialog`，hooks 为 `useAuth`、`usePagination`。

### 技术与规模要求

前端 Angular 17 + TypeScript + Vite；后端 Go 1.22 + Gin + GORM；PostgreSQL + Redis + MinIO。目标 3000–4200 行、30–42 个 `.go` 文件。

### 文件结构强制清单

前端 `api/stores/types/components/common/hooks/pages/router/utils`；后端 `model/dto/repository/service/handler/router/middleware/constants/util`，README 列明共享枚举位置。

### 结构红线

严禁合并职责到单一文件；资质、清单、核验和审计必须拆分实现。

### 部署与交付

根目录必须提供 `docker-compose.yml`（顶层 `name: hazardous-waste-transfer-compliance`，且不写 `version:`）、`.env` 和 `.env.example`（均含 `COMPOSE_PROJECT_NAME=hazardous-waste-transfer-compliance`）、`README.md`、`frontend/Dockerfile`、`backend/Dockerfile` 和 `frontend/nginx.conf`。前端端口 `18518`、后端端口 `19518`；Nginx `/api` 代理、数据库健康检查、命名卷和 `condition: service_healthy` 齐全，提供真实 `/healthz`、Git 初始化。
