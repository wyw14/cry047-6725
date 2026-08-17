# 公共设施预防性保养协同平台

> 公共设施预防性保养协同平台 baseline。图书馆、展馆等场所的设施经常只登记负责人，
> 周期保养和异常证据缺乏连续记录；本系统根据周期生成保养安排、发现逾期并跟踪恢复。

本仓库是一个**干净的 baseline**：没有故意插入的 bug，所有领域规则都按源 prompt 落地，
全部命令可在离线环境（无需 PostgreSQL / 第三方接口 / CDN / 在线模型）下验证通过。

---

## 模块职责

```
cry047-baseline/
├── cmd/server/                 # 程序入口：组装 ports、启动 HTTP、优雅停机
├── internal/
│   ├── config/                 # 环境变量驱动的配置 (Default / FromEnv)
│   ├── domain/                 # 领域模型、错误码、Ports 接口、状态机校验
│   ├── application/             # 应用服务（用例编排、审计、通知、时间线）
│   ├── repository/
│   │   ├── memory/             # 默认的离线内存仓储实现
│   │   └── postgres/           # PostgreSQL 仓储实现 (生产可选)
│   ├── service/
│   │   ├── notifier/           # 本地通知适配器（环形缓冲 + 可选日志）
│   │   ├── scheduler/          # 本地调度器（goroutine + ticker）
│   │   └── storage/            # 本地附件存储（白名单 + 内容寻址）
│   ├── transport/
│   │   └── http/               # Gin 路由、统一响应、错误映射
│   │       └── handler/        # 各资源的 HTTP handler
│   ├── middleware/             # request_id、CORS、安全头、panic 恢复、审计、RBAC
│   └── platform/
│       ├── logger/             # zap 结构化日志
│       ├── postgres/           # pgx 连接池封装
│       └── validator/          # go-playground/validator 单例
├── migrations/                 # 0001_init.up/down.sql，幂等可重复执行
├── api/openapi/                # OpenAPI 3.0 规范
├── scripts/seed/               # 演示数据（幂等，不覆盖已有数据）
├── tests/                      # 跨包集成测试（build tag: integration）
├── web/                        # Vue 3 + TypeScript + Vite + Pinia 前端
├── .env.example                # 环境变量样例
├── .gitignore
├── Makefile
├── go.mod / go.sum
└── README.md
```

**领域分层约束**：业务规则不堆在 `main.go` / handler / 单个 service 文件里。状态机校验
落在 `domain`；跨聚合的事务/校验、审计、时间线和通知由 `application` 服务编排；HTTP
层只做参数绑定、调用服务、写响应。

---

## 本地启动

### 1. 安装依赖

```bash
# 后端
go mod tidy

# 前端
cd web && npm install
```

### 2. 直接运行（默认离线）

```bash
# 默认使用内存仓储，开启时自动 seed 演示数据
go run ./cmd/server
# 或
make run
```

服务监听 `:8080`，健康检查：

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

### 3. 通过环境变量定制

```bash
cp .env.example .env
# 编辑 .env 后：
set -a; . ./.env; set +a
go run ./cmd/server
```

或在 PowerShell / cmd 中按需 `$env:HTTP_ADDR=":9090"` 等单独覆盖。

---

## 配置

所有配置项见 `.env.example`。关键项：

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | HTTP 监听地址 |
| `HTTP_MODE` | `release` | gin 模式 (`release`/`test`/`debug`) |
| `USE_MEMORY_STORE` | `true` | 默认离线内存仓储；置 `false`（同时设置 `DATABASE_URL`）切换到 PostgreSQL |
| `DATABASE_URL` | 空 | PostgreSQL DSN，设置后自动 `USE_MEMORY_STORE=false` |
| `STORAGE_ROOT` | `./var/attachments` | 附件存储根目录 |
| `STORAGE_MAX_SIZE` | `8388608` (8MiB) | 单文件最大字节数 |
| `NOTIFIER_RING_SIZE` | `256` | 本地通知环形缓冲容量 |
| `SCHEDULER_TICK` | `1m` | 调度器扫描间隔 |
| `ALLOWED_ORIGINS` | `http://localhost:5173,...` | CORS 白名单 |
| `SEED_ON_START` | `true` | 启动时若内存仓储为空则自动 seed 演示数据 |

---

## 迁移

`migrations/0001_init.up.sql` 与 `migrations/0001_init.down.sql` 完全幂等
（`CREATE TABLE IF NOT EXISTS` / `DROP TABLE IF EXISTS`），可重复执行。

```bash
# 用 psql 手工执行
psql "$DATABASE_URL" -f migrations/0001_init.up.sql

# 回滚
psql "$DATABASE_URL" -f migrations/0001_init.down.sql
```

集成测试 `tests/postgres_integration_test.go` 也覆盖了迁移的 apply / rollback
（需 `-tags integration` 且 `DATABASE_URL` 已配置，详见下文）。

---

## 演示数据

`scripts/seed/seed.go` 在启动时若 `SEED_ON_START=true` 且内存仓储为空，会插入：

- 3 个场所：市中心图书馆、城市博物馆、国际展览中心
- 3 个责任人：张伟、李娜、王强
- 4 个设施：主空调机组（critical）、1 号电梯（important）、消防控制柜（critical）、展厅照明系统（standard）
- 4 个保养模板：空调月度保养、电梯季度保养、消防周检、照明半年保养
- 4 个保养计划：覆盖每个设施，部分已逾期以便触发调度器行为

种子数据**幂等**：以业务主键为唯一键，已存在的记录会被跳过，不会覆盖。

---

## 接口示例

所有接口位于 `/api/v1` 前缀下。完整规范见 `api/openapi/openapi.yaml`。

### 创建设施

```bash
curl -X POST http://localhost:8080/api/v1/facilities \
  -H "Content-Type: application/json" \
  -H "X-Actor-Id: admin-1" -H "X-Actor-Name: 管理员" -H "X-Actor-Role: admin" \
  -d '{
    "place_id":"place-lib-001",
    "name":"新风机组",
    "code":"FRESH-001",
    "category":"空调",
    "responsible_person_id":"rp-001",
    "criticality":"important",
    "description":"图书馆新风"
  }'
```

### 列表 + 分页 + 排序 + 白名单筛选

```bash
curl 'http://localhost:8080/api/v1/facilities?limit=10&offset=0&order_by=name&order=asc&criticality=critical'
```

### 设施状态转换

```bash
curl -X POST http://localhost:8080/api/v1/facilities/fac-hvac-001/transition \
  -H "Content-Type: application/json" \
  -H "X-Actor-Role: operator" \
  -d '{"to":"pending_maintenance","reason":"计划保养"}'
```

### 异常登记 → 整改 → 复检 → 恢复确认

```bash
# 1. 登记异常
curl -X POST http://localhost:8080/api/v1/anomalies \
  -H "Content-Type: application/json" -H "X-Actor-Role: operator" \
  -d '{
    "facility_id":"fac-hvac-001",
    "discovered_at":"2026-08-17T10:00:00Z",
    "description":"机组异响严重",
    "severity":"critical",
    "idempotency_key":"anom-1"
  }'

# 2. 整改 (engineer/supervisor/admin)
curl -X POST http://localhost:8080/api/v1/anomalies/<id>/rectify \
  -H "Content-Type: application/json" -H "X-Actor-Role: supervisor" \
  -d '{"measure":"更换轴承并校准"}'

# 3. 复检
curl -X POST http://localhost:8080/api/v1/anomalies/<id>/reinspect \
  -H "Content-Type: application/json" -H "X-Actor-Role: supervisor" \
  -d '{"result":"复检通过","pass":true}'

# 4. 恢复确认 (supervisor/admin)
curl -X POST http://localhost:8080/api/v1/anomalies/<id>/recover \
  -H "X-Actor-Role: supervisor"
```

### 触发调度器扫描

```bash
curl -X POST http://localhost:8080/api/v1/scheduler/scan -H "X-Actor-Role: admin"
```

---

## 状态规则

### 设施状态机

允许的转换：

| from | to |
| --- | --- |
| `normal` | `pending_maintenance`, `restricted_use`, `under_repair` |
| `pending_maintenance` | `normal`, `overdue`, `restricted_use`, `under_repair` |
| `overdue` | `under_repair`, `restricted_use`, `recovered` |
| `restricted_use` | `under_repair`, `recovered` |
| `under_repair` | `recovered` |
| `recovered` | `normal`, `pending_maintenance` |

**关键不变量**：

- **逾期关键设施不能被直接标记为正常**：`overdue -> normal` 对 `critical` 设施被拒绝，
  必须经过 `under_repair -> recovered -> normal` 完整流程（`domain/facility.go`）。
- **限用关键设施同理**：`restricted_use -> normal` 对 `critical` 设施也被拒绝。
- 状态未变化（`from == to`）直接返回 `STATE_FORBIDDEN`。

### 异常生命周期

`open -> rectifying/reinspecting -> recovered`，可选 `closed_no_action`。
恢复前必须经过复检；只有 `recovered` 状态的异常才能被"恢复确认"。

### 计划周期变更

**更换周期只影响后续计划，不能篡改已完成记录**：
`PlanService.ChangeCycle` 只更新 `MaintenancePlan.CurrentCycleDays` / `NextDueDate`，
**从不修改任何 `Execution` 记录**。校验在 `application/plan_service.go` 与对应单元测试中。

### 幂等键

`executions` / `anomalies` / `todos` 的 `IdempotencyKey` 唯一，重复提交返回首次创建的
记录而不是新建。仓储层（内存 / PostgreSQL）都强制唯一约束。

### 乐观并发

所有 `Update*` 都按 `version` 字段做乐观并发控制；版本不匹配返回 `CONFLICT`。
内存仓储的并发测试见 `internal/repository/memory/concurrency_test.go`。

---

## 测试命令

```bash
# Go 侧
go mod tidy
go build ./...
go test ./...
go test -race ./...
go vet ./...
gofmt -l .

# 前端侧
cd web
npm install
npm run test -- --run
npm run build

# 顶部便利脚本
make test test-race vet fmt-check
make frontend-test frontend-build
```

### 集成测试（可选，需要 PostgreSQL）

```bash
# 假设本地已起 PostgreSQL，导入迁移
psql "postgres://user:pass@localhost:5432/maintenance?sslmode=disable" \
  -f migrations/0001_init.up.sql

# 运行带 integration tag 的测试
DATABASE_URL="postgres://user:pass@localhost:5432/maintenance?sslmode=disable" \
  go test -tags integration ./tests/...
```

未设置 `DATABASE_URL` 时集成测试会自动 `t.Skip`，所以 `go test ./...` 始终是离线自包含的。

---

## 实际验证结果

本 baseline 在以下环境实际执行并通过（Windows 10 / Go 1.26.2 / Node 22.x）：

```text
$ go version
go version go1.26.2 windows/amd64

$ gofmt -l .
(empty)

$ go mod tidy && go build ./...
(no output; exit 0)

$ go vet ./...
(no output; exit 0)

$ go test ./...
ok  	github.com/cry047/baseline/internal/application
ok  	github.com/cry047/baseline/internal/domain
ok  	github.com/cry047/baseline/internal/repository/memory
ok  	github.com/cry047/baseline/internal/service/notifier
ok  	github.com/cry047/baseline/internal/service/scheduler
ok  	github.com/cry047/baseline/internal/service/storage
ok  	github.com/cry047/baseline/internal/transport/http

$ go test -race ./...
(same packages, all ok with -race)

$ cd web && npm install && npm run test -- --run && npm run build
(vitest: passed; vite build: dist/ generated)
```

测试覆盖：

- 领域单元测试：状态机、输入校验、RBAC 角色 (`internal/domain/*_test.go`)
- 应用服务测试：异常全流程、计划周期不篡改历史、计划跳过/重新生成、调度器逾期标记
  与待办生成幂等性 (`internal/application/*_test.go`)
- HTTP 校验与权限测试：分页、CORS、安全头、状态转换拒绝、角色边界、并发提交
  (`internal/transport/http/http_test.go`)
- 仓储并发与事务测试：乐观并发、幂等键冲突 (`internal/repository/memory/concurrency_test.go`)
- 附件存储测试：路径穿越拒绝、符号链接拒绝、内容寻址、原子改名
  (`internal/service/storage/local_storage_test.go`)
- 调度器测试：回调触发、取消、时钟注入 (`internal/service/scheduler/local_scheduler_test.go`)
- 通知器测试：环形缓冲容量限制、最近通知 (`internal/service/notifier/local_notifier_test.go`)

---

## 安全 / 合规

- 不提交密钥、依赖缓存、构建产物或运行期数据（见 `.gitignore`）
- 上传类型与大小校验（白名单 MIME + 扩展名 + 字节数上限）
- 敏感字段脱敏：审计日志中不记录密码 / token；未来扩展点已留好 `domain.Actor` 抽象
- 安全响应头：`X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`、
  `Strict-Transport-Security`、`Content-Security-Policy`、`X-XSS-Protection`
- CORS 白名单严格匹配 Origin
- panic 恢复：所有 HTTP 请求被 `middleware.PanicRecovery` 包裹，panic 返回 500 + request_id
- 优雅停机：捕获 SIGINT/SIGTERM，调用 `srv.Shutdown(ctx)`，等待 inflight 请求结束

---

## License

本 baseline 仅用于内部演示与评测。
