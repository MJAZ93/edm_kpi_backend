# Implementation Plan — KPI Platform Backend

Ordered by dependency. Each phase is self-contained and testable before moving on.

---

## Phase 1 — Project Bootstrap

**Goal:** Runnable Gin server with DB connection, swagger, and CORS.

### Tasks
- [ ] `go mod init kpi-backend` and add all dependencies
- [ ] Create `.env` and `.env.example`
- [ ] `main.go` — LoadEnv → LoadDatabase → ServeApplication
- [ ] `app/app.go` — env load, DB connect, AutoMigrate stub, start server
- [ ] `app/route.go` — PublicRoutes, PrivateRoutes, Swagger stubs
- [ ] `dao/main.go` — Connect() with PostGIS extension + spatial index setup
- [ ] `middleware/default_mdw.go` — CORS + OPTIONS
- [ ] `middleware/jwt_mdw.go` — JWT validation
- [ ] `middleware/role_mdw.go` — role guard
- [ ] `util/jwt.go` — GenerateJWT, ValidateJWT, ExtractUserID, ExtractRole
- [ ] `util/pagination.go` — ParsePagination, PaginatedResponse
- [ ] Verify: server starts, `/swagger/index.html` loads

---

## Phase 2 — Auth & Users

**Goal:** Login, password reset, user CRUD.

### Models
- [ ] `model/user.go` — User struct + BeforeSave bcrypt hook

### DAOs
- [ ] `dao/user_dao.go`
  - Create, GetByID, GetByEmail, List(page, limit), Update, SoftDelete
  - GetByEmailAndPassword (bcrypt compare)
  - SetPasswordResetToken, GetByResetToken, ClearResetToken

### Controllers + Services
- [ ] `controller/auth_controller.go` — Login, ForgotPassword, ResetPassword
- [ ] `service/auth_service.go` — route registration + swagger annotations
- [ ] `controller/user_controller.go` — Me, List, Create, Single, Update, Delete
- [ ] `service/user_service.go`

### Utils
- [ ] `util/password.go` — HashPassword, CheckPassword (thin wrappers over bcrypt)
- [ ] `util/email.go` — SendEmail + `password_reset` template + `welcome` template

### Routes registered
```
POST /public/auth/login
POST /public/auth/forgot-password
POST /public/auth/reset-password
GET  /private/users          [CA]
POST /private/users          [CA]
GET  /private/users/me       [All]
GET  /private/users/:id      [CA]
PUT  /private/users/:id      [CA]
DELETE /private/users/:id    [CA]
```

---

## Phase 3 — Organisation Structure

**Goal:** Full org hierarchy CRUD: Pelouro → Direção → Departamento.

### Models
- [ ] `model/pelouro.go`
- [ ] `model/direcao.go`
- [ ] `model/departamento.go` — with `DepartamentoUser` join table

### DAOs
- [ ] `dao/pelouro_dao.go` — CRUD
- [ ] `dao/direcao_dao.go` — CRUD, ListByPelouro
- [ ] `dao/departamento_dao.go` — CRUD, ListByDirecao, AddUser, RemoveUser, GetUsers

### Controllers + Services
- [ ] pelouro, direcao, departamento controllers + services
- [ ] `controller/departamento_controller.go` — including add/remove user endpoints

### Special endpoint
- [ ] `GET /private/org/tree` — single query that builds full nested JSON tree
  - Hint: use GORM Preload chain or raw SQL with JSON aggregation

### AutoMigrate additions in app.go

---

## Phase 4 — Geography

**Goal:** Region/ASC CRUD with PostGIS polygon support.

### Models
- [ ] `model/geo.go` — Regiao struct (polygon as string), ASC struct

### DAOs
- [ ] `dao/geo_dao.go`
  - CreateRegiao, UpdateRegiao, UpdateRegiaoPolygon (raw SQL for ST_GeomFromGeoJSON)
  - GetRegiaoWithPolygon (raw SQL for ST_AsGeoJSON)
  - Same pattern for ASC
  - GetAllWithGeoJSON — for map endpoint

### Controllers + Services
- [ ] `controller/geo_controller.go`
- [ ] `service/geo_service.go`

### Routes
```
GET/POST/GET/:id/PUT/:id/DELETE/:id for /private/geo/regioes and /private/geo/ascs
```

---

## Phase 5 — Projects

**Goal:** Hierarchical projects (self-referencing) with visibility scoping.

### Models
- [ ] `model/project.go`

### DAOs
- [ ] `dao/project_dao.go`
  - Create, GetByID, List (with role-based filter), Update, SoftDelete
  - GetChildren(parent_id)
  - GetTree(id) — recursive CTE or iterative loader

### Controllers + Services
- [ ] `controller/project_controller.go`
- [ ] `service/project_service.go`

### Business logic
- Visibility: CA sees all; PELOURO sees projects where `creator_org_id` matches their pelouro or below; etc.

---

## Phase 6 — Tasks & Milestones

**Goal:** Core KPI tracking: tasks with goals, milestones with photos, score recalculation.

### Models
- [ ] `model/task.go` — Task + TaskScope
- [ ] `model/milestone.go`

### DAOs
- [ ] `dao/task_dao.go`
  - CRUD, ListByProject, GetWithScopes
  - RecalcCurrentValue(task_id) — UPDATE tasks SET current_value = (SELECT COALESCE(SUM(achieved_value),0) FROM milestones WHERE task_id=? AND status NOT IN ('BLOCKED') AND deleted_at IS NULL)
  - UpdateNextUpdateDue(task_id) — based on frequency
- [ ] `dao/milestone_dao.go` — CRUD, ListByTask

### Utils
- [ ] `util/storage.go` — UploadPhoto (local disk first; S3 branch gated by env)

### Controllers + Services
- [ ] `controller/task_controller.go`
- [ ] `service/task_service.go`
- [ ] `controller/milestone_controller.go` — including photo upload endpoint
- [ ] `service/milestone_service.go`

### On milestone PUT: trigger chain (see Phase 8 for notifications)

---

## Phase 7 — Blockers

**Goal:** Impedimento reporting + SLA auto-approval.

### Models
- [ ] `model/blocker.go`

### DAOs
- [ ] `dao/blocker_dao.go`
  - Create (auto-compute `auto_approve_at = created_at + sla_days * 24h`)
  - GetByEntity, Approve, Reject
  - ListPendingExpired — for background job

### Controllers + Services
- [ ] `controller/blocker_controller.go`
- [ ] `service/blocker_service.go`

### Background job (in jobs/scheduler.go)
- Hourly: fetch expired pending blockers → set AUTO_APPROVED → notify

---

## Phase 8 — Audit + Notifications

**Goal:** Full notification chain and audit trail.

### Models
- [ ] `model/audit_log.go`
- [ ] `model/notification.go`

### DAOs
- [ ] `dao/audit_dao.go` — Write, ListByEntity
- [ ] `dao/notification_dao.go` — Create, ListByUser, MarkRead, MarkAllRead

### Utils
- [ ] `util/email.go` — add all templates: `task_updated`, `blocker_created`, `blocker_resolved`, `forecast_risk`, `milestone_overdue`

### Notification service
- [ ] `service/notification_service.go`
  - `NotifyChain(taskID, actorID)` — walk up org hierarchy, create Notification + send email
  - `NotifyASCDirector(taskID, actorID)` — for scoped tasks
  - `NotifyBlockerReporter(blockerID, status)` — approval/rejection
  - `NotifyForecastRisk(taskID)` — forecast alert

### Middleware
- [ ] `middleware/audit_mdw.go` — intercept POST/PUT/DELETE, write audit log after handler completes

### Controllers + Services
- [ ] `controller/notification_controller.go`
- [ ] `service/notification_service.go` (route registration)
- [ ] `controller/audit_controller.go`
- [ ] `service/audit_service.go`

---

## Phase 9 — Performance Scoring & Cache

**Goal:** Compute and cache scores for all entity types.

### Models
- [ ] `model/performance_cache.go`

### Utils
- [ ] `util/score.go`
  - `ComputeExecutionScore(tasks []Task, milestones []Milestone) float64`
  - `ComputeGoalScore(task Task) float64`
  - `ComputePerformanceScore(exec, goal float64) float64`
  - `GetTrafficLight(score float64) string`

### DAO
- [ ] `dao/performance_dao.go`
  - `RefreshCacheForTask(taskID)` — recomputes all affected entities up the chain
  - `GetScore(entityType, entityID, period)` — read from cache
  - `RefreshAllForPeriod(period)` — full refresh (nightly job)

### Background job
- [ ] Nightly cache refresh in `jobs/scheduler.go`

---

## Phase 10 — Dashboard & Analytics

**Goal:** All analytical endpoints.

### DAOs
- [ ] `dao/dashboard_dao.go`
  - Summary query
  - DrillDown (aggregate scores per child entities)
  - GetMapData (GeoJSON with scores)
  - GetTopPerformers
  - GetTimeline
  - GetDistribution
  - GetBenchmark

### Utils
- [ ] `util/forecast.go` — ForecastTask(task Task) → ForecastResult

### Controller + Service
- [ ] `controller/dashboard_controller.go`
- [ ] `service/dashboard_service.go`

---

## Phase 11 — Background Jobs (Scheduler)

**Goal:** All background goroutines running reliably.

- [ ] `jobs/scheduler.go` — `StartScheduler(db *gorm.DB)`
  - Blocker SLA job (every 1h)
  - Nightly performance cache refresh (00:00)
  - Daily forecast alert (07:00)
  - Daily milestone overdue check (08:00) — notify responsible if milestone past `planned_date` and still PENDING

---

## Phase 12 — Integration Tests

**Goal:** Full test coverage for critical paths using SQLite in-memory.

- [ ] Auth: login, forgot password, reset password
- [ ] Project/Task/Milestone CRUD
- [ ] Score computation (unit tests for util/score.go)
- [ ] Forecast computation (unit tests for util/forecast.go)
- [ ] Role access control (403 for wrong roles)
- [ ] Blocker auto-approval logic

---

## Summary: Files to Create

| Directory | Files (count) |
|-----------|--------------|
| model/ | 12 files |
| dao/ | 14 files |
| controller/ | 13 files |
| service/ | 13 files |
| middleware/ | 4 files |
| util/ | 7 files |
| jobs/ | 1 file |
| app/ | 2 files |
| root | main.go, .env, .env.example, go.mod |
| **Total** | **~71 files** |

---

## Execution Order (when ready to code)

```
Phase 1  → Phase 2  → Phase 3  → Phase 4
                                    ↓
Phase 5 (projects) ← Phase 4 (geography depends on same DB)
    ↓
Phase 6 (tasks + milestones)
    ↓
Phase 7 (blockers)
    ↓
Phase 8 (audit + notifications) ← depends on tasks & blockers
    ↓
Phase 9 (scoring) ← depends on milestones
    ↓
Phase 10 (dashboard) ← depends on scoring
    ↓
Phase 11 (jobs) ← depends on scoring, notifications, blockers
    ↓
Phase 12 (tests)
```
