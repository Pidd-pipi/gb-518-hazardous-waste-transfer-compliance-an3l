# 验证记录

验证日期：2026-08-22（Asia/Shanghai）

## 静态检查与构建

| 检查 | 结果 |
|---|---|
| `gofmt -w cmd internal` | 通过 |
| `go test ./...` | 通过 |
| `go test -race ./...` | 通过，包含真实 Gin + SQLite 路由集成测试 |
| `go vet ./...` | 通过 |
| `go build ./...` | 通过 |
| `npm run typecheck` | 通过 |
| `npm run build` | 通过 |
| `docker compose config --quiet` | 通过 |

后端非测试代码实测为 3269 行、38 个 `.go` 文件，符合提示词要求的 3000–4200 行、30–42 文件。

## 空卷 Compose 与 API

执行：

```bash
KEEP_RUNNING=1 ./scripts/validate.sh
```

脚本先执行 `docker compose down -v --remove-orphans`，随后创建新的 PostgreSQL、Redis、MinIO 命名卷并重新构建全部服务。实测服务：

| 服务 | 宿主端口 | 结果 |
|---|---:|---|
| frontend | 18518 | 启动成功，Nginx `/api` 代理可用 |
| backend | 19518 | healthy，`/healthz` 返回 database/redis ready |
| PostgreSQL | 20518 | healthy |
| Redis | 21518 | healthy |
| MinIO API / Console | 22518 / 23518 | healthy |

自动化断言均通过：

- viewer 能读取四类业务数据，写联单和读取审计均返回 403。
- operator 创建关联 `WG-001`、`CP-002` 的联单成功，request ID 为 `validation-manifest-create`。
- `draft → in_transit` 跳级返回 422；合法 `draft → submitted` 成功并写入 `validation-manifest-submit`。
- 使用旧版本推进联单返回 409。
- 关联尚未核准的 `CP-001` 时，联单提交返回 422。
- operator 创建核验成功，但作出通过决定返回 403。
- reviewer 执行 `pending → pass` 成功，决定依据持久化，request ID 为 `validation-reviewer-decision`。
- 审计汇总包含至少 5 条写操作和至少 2 次状态迁移；自定义 request ID 可从审计列表检索。

## 内置 Browser 验收

严格使用 Codex 内置 Browser，未使用外部 Chrome。桌面视口为 1440×900，移动视口为 390×844。

| 页面 | 实测内容 | 结果 |
|---|---|---|
| `/generators` | 许可编号、有效期、废物类别、共享 `LicensePanel`、分支状态按钮 | 通过 |
| `/carriers` | 许可证、有效期、车辆数、共享 `LicensePanel`、复核按钮 | 通过 |
| `/manifests` | 产废/承运关联、废物重量、提交/发运/签收/驳回分支 | 通过 |
| `/checks` | 联单关联、决定依据、终态锁定、新增与状态确认弹窗 | 通过 |
| `/audit` | actor、action、实体、前后状态和 request ID | 通过 |

关键交互实测：

- 登录页通过真实 `/api/auth/login` 进入管理员工作台。
- 五个导航链接均在页面内切换并更新正确主内容。
- 在核验页创建 `COMPLIANCECHECK-335894`，随后通过确认弹窗执行 `pending → pass`。
- 审计页立即出现对应的 `admin · create` 与 `admin · transition`，两条记录使用不同 UUID request ID。
- 桌面截图中导航、许可摘要、指标、表格和审计列表无重叠或截断。
- 移动端文档宽度与视口一致，页面级横向溢出为 0；联单表格使用 1120px 自身滚动区，名称列实测 190px，不再逐字竖排。
- 最终浏览器控制台 error/warning 数量为 0。

Browser 验收过程中发现并修复了三项真实问题：登录完成后根视图未刷新、URL 更新但 `RouterOutlet` 保留旧页面、移动端表格列宽过度压缩。修复后均重新构建并复验通过。

## 清理与 Git

已执行：

```bash
docker compose down -v --remove-orphans
git init -b main
git config user.name gaobo
git config user.email gaobo@benzhi.io
```

结果：Compose 项目容器列表和命名卷列表均为空；仓库分支为 `main`，本地身份为 `gaobo <gaobo@benzhi.io>`。未创建提交，未推送远端。
