# Architecture — KPI Platform Backend

## Stack (follows go_init.md conventions)

| Layer | Technology |
|-------|-----------|
| HTTP Router | Gin v1.10 |
| ORM | GORM v1.25 + gorm/driver/postgres |
| DB | PostgreSQL 15 + PostGIS |
| Auth | JWT (golang-jwt/v4) |
| Docs | Swagger (swaggo) |
| Config | godotenv |
| Email | net/smtp (standard lib) or `github.com/wneessen/go-mail` |
| File Storage | local disk / AWS S3 (`aws-sdk-go-v2`) |
| Background Jobs | time.Ticker goroutines (simple cron-like) |

---

## Project Structure

```
backend/
├── .env
├── .env.example
├── go.mod
├── go.sum
├── main.go
├── app/
│   ├── app.go          # LoadEnv, LoadDatabase, ServeApplication
│   └── route.go        # PublicRoutes, PrivateRoutes, Swagger
├── controller/
│   ├── auth_controller.go
│   ├── user_controller.go
│   ├── pelouro_controller.go
│   ├── direcao_controller.go
│   ├── departamento_controller.go
│   ├── geo_controller.go
│   ├── project_controller.go
│   ├── task_controller.go
│   ├── milestone_controller.go
│   ├── blocker_controller.go
│   ├── dashboard_controller.go
│   ├── notification_controller.go
│   └── audit_controller.go
├── service/
│   ├── auth_service.go
│   ├── user_service.go
│   ├── pelouro_service.go
│   ├── direcao_service.go
│   ├── departamento_service.go
│   ├── geo_service.go
│   ├── project_service.go
│   ├── task_service.go
│   ├── milestone_service.go
│   ├── blocker_service.go
│   ├── dashboard_service.go
│   ├── notification_service.go
│   └── audit_service.go
├── dao/
│   ├── main.go             # DB connection (Connect())
│   ├── user_dao.go
│   ├── pelouro_dao.go
│   ├── direcao_dao.go
│   ├── departamento_dao.go
│   ├── geo_dao.go
│   ├── project_dao.go
│   ├── task_dao.go
│   ├── milestone_dao.go
│   ├── blocker_dao.go
│   ├── dashboard_dao.go
│   ├── notification_dao.go
│   ├── audit_dao.go
│   └── performance_dao.go
├── model/
│   ├── user.go
│   ├── pelouro.go
│   ├── direcao.go
│   ├── departamento.go
│   ├── geo.go              # Regiao, ASC
│   ├── project.go
│   ├── task.go             # Task, TaskScope
│   ├── milestone.go
│   ├── blocker.go
│   ├── audit_log.go
│   ├── notification.go
│   └── performance_cache.go
├── middleware/
│   ├── default_mdw.go      # CORS + OPTIONS
│   ├── jwt_mdw.go          # JWT validation
│   ├── role_mdw.go         # role-based access guard
│   └── audit_mdw.go        # automatic audit log on write ops
├── util/
│   ├── jwt.go              # GenerateJWT, ValidateJWT
│   ├── password.go         # Hash, Compare
│   ├── email.go            # SendEmail, templates
│   ├── storage.go          # UploadPhoto (local / S3)
│   ├── score.go            # ComputePerformanceScore, TrafficLight
│   ├── forecast.go         # ForecastTask
│   └── pagination.go       # PaginationParams, PaginatedResponse
├── jobs/
│   └── scheduler.go        # Background jobs (blocker SLA, cache refresh, forecast alerts)
├── uploads/                # local file storage (gitignored)
└── docs/                   # swag-generated + design docs
    ├── 01_schema.md
    ├── 02_api_spec.md
    ├── 03_architecture.md
    └── 04_implementation_plan.md
```

---

## Layer Responsibilities

### controller/
- Bind & validate HTTP request (ShouldBindJSON / ShouldBindQuery)
- Call DAO or service methods
- Return JSON response
- No business logic; no direct DB calls

### service/
- Register routes on `*gin.RouterGroup` with Swagger annotations
- Light orchestration when a flow needs multiple DAO calls
- Holds `Route string` and `Controller` reference (per go_init.md pattern)

### dao/
- All GORM queries
- Accepts plain Go values / structs; returns domain models
- No HTTP context; no business rules

### model/
- GORM model structs + hooks (BeforeSave for password hashing)
- JSON tags for response serialisation
- `gorm:"column:..."` tags where column name diverges from field name

### middleware/
- `DefaultAuthMiddleware` — CORS headers, OPTIONS preflight
- `JWTAuthMiddleware` — validate bearer token, inject `user_id` + `role` into context
- `RoleMiddleware(roles ...string)` — accept list of allowed roles, abort 403 otherwise
- `AuditMiddleware` — for PUT/DELETE/POST on write-sensitive routes: capture before/after and write to `audit_logs` (done after response)

### util/
- **jwt.go** — GenerateJWT, ValidateJWT, ExtractUserID(c), ExtractRole(c)
- **password.go** — HashPassword, CheckPassword
- **email.go** — SendEmail(to, subject, body); template functions per notification type
- **storage.go** — UploadPhoto(file) → returns URL; reads `STORAGE_BACKEND` env
- **score.go** — ComputeExecutionScore, ComputeGoalScore, ComputePerformanceScore, GetTrafficLight
- **forecast.go** — ForecastTask(task) → ForecastResult
- **pagination.go** — ParsePagination(c), PaginatedResponse(data, total, page, limit)

### jobs/
Goroutines started in `app.ServeApplication()` before `r.Run()`:

```go
jobs.StartScheduler(dao.Database)
```

Three jobs running in separate goroutines:

1. **Blocker SLA auto-approval** — every hour
   - `SELECT * FROM blockers WHERE status='PENDING' AND auto_approve_at <= NOW()`
   - Update to `AUTO_APPROVED`, log audit, send notification

2. **Performance cache refresh** — every night at 00:00
   - Recompute scores for all entities for current month
   - Upsert into `performance_cache`

3. **Forecast alert** — every day at 07:00
   - For all ACTIVE tasks with end_date > today
   - Compute forecast; if `projected_final < target * 0.9` → create FORECAST_RISK notification if not already sent today

---

## Authentication & Authorization Flow

```
Request
  │
  ▼
DefaultAuthMiddleware (CORS)
  │
  ▼
[Public routes] ─────────────────────────► Controller (no auth)
  │
JWTAuthMiddleware
  │ (extracts user_id + role, stores in gin.Context)
  ▼
RoleMiddleware(allowed roles...)
  │ (403 if role not in list)
  ▼
Controller
```

### Extracting current user in controllers:
```go
userID := c.GetUint("user_id")
role   := c.GetString("role")
```

### Scoped visibility
Each controller/DAO applies visibility scoping based on role:
- **CA** → sees everything
- **PELOURO** → sees their pelouro + below
- **DIRECAO** → sees their direção + below
- **DEPARTAMENTO** → sees only their departamento

---

## Notification Flow

When a milestone is updated (PUT /private/milestones/:id):

```
1. MilestoneDAO.Update(milestone)
2. TaskDAO.RecalcCurrentValue(task_id)   ← sum achieved_value
3. PerformanceDAO.RefreshCacheForTask(task_id)
4. NotificationService.NotifyChain(task_id, actor_user_id)
   ├─ Resolve chain: Departamento → Direção → Pelouro → CA
   ├─ Create Notification records for each superior
   └─ util.email.Send(...) for each
5. If task has ASC scope → notify ASC director
6. AuditLog.Write(entity=milestone, action=UPDATE, before/after)
```

---

## Email Templates

Located in `util/email.go` as Go template strings:

| Template | Trigger |
|----------|---------|
| `task_updated` | Task or milestone updated |
| `blocker_created` | Blocker reported on your scope |
| `blocker_approved` | Blocker approved/rejected |
| `forecast_risk` | Task at risk of missing goal |
| `milestone_overdue` | Milestone not updated by due date |
| `password_reset` | Forgot password flow |
| `welcome` | New user created |

---

## .env Variables

```env
# Server
IP=0.0.0.0
PORT=8000
SCHEME=http
APP_NAME=KPI Platform

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=kpi_db

# Auth
JWT_PRIVATE_KEY=CHANGE_ME_SUPER_SECRET
TOKEN_TTL=86400

# Email (SMTP)
SMTP_HOST=smtp.edm.co.mz
SMTP_PORT=587
SMTP_USER=noreply@edm.co.mz
SMTP_PASSWORD=
SMTP_FROM=noreply@edm.co.mz

# Storage
STORAGE_BACKEND=local          # local | s3
UPLOAD_DIR=./uploads
# S3 (only if STORAGE_BACKEND=s3)
AWS_BUCKET=kpi-uploads
AWS_REGION=eu-west-1
AWS_ACCESS_KEY=
AWS_SECRET_KEY=

# Performance cache
SCORE_EXECUTION_WEIGHT=0.6
SCORE_GOAL_WEIGHT=0.4
BLOCKER_SLA_DEFAULT_DAYS=3
```

---

## go.mod Dependencies

```
module kpi-backend

go 1.22

require (
  github.com/gin-gonic/gin                v1.10.0
  github.com/golang-jwt/jwt/v4            v4.5.0
  github.com/joho/godotenv                v1.5.1
  github.com/swaggo/files                 v1.0.1
  github.com/swaggo/gin-swagger           v1.6.0
  gorm.io/driver/postgres                 v1.5.9
  gorm.io/gorm                            v1.25.7
  golang.org/x/crypto                     v0.22.0
  github.com/google/uuid                  v1.6.0
  github.com/wneessen/go-mail             v0.4.1   // SMTP
  github.com/aws/aws-sdk-go-v2            v1.27.0  // S3 (optional)
  github.com/aws/aws-sdk-go-v2/service/s3 v1.54.0
)
```

PostGIS geometry handling: store polygon as `string` (GeoJSON text) in GORM model; convert to/from PostGIS with raw SQL when querying map data:
```go
// Insert
db.Exec("UPDATE regioes SET polygon = ST_GeomFromGeoJSON(?) WHERE id = ?", geojsonStr, id)

// Select for GeoJSON API
db.Raw("SELECT id, name, ST_AsGeoJSON(polygon) as polygon_geojson FROM regioes").Scan(&rows)
```

---

## Swagger Setup

1. `go install github.com/swaggo/swag/cmd/swag@latest`
2. Annotations in `main.go`:
```go
// @title           KPI Platform API
// @version         1.0
// @description     Backend for KPI management platform
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
```
3. Per-route annotations in `service/*.go` files
4. `swag init` → regenerates `docs/`
5. Access: `http://localhost:8000/swagger/index.html`
