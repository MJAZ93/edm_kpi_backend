# API Specification — KPI Platform

Base path: `/api/v1`
Auth: JWT Bearer token on all `/private/*` routes
Content-Type: `application/json`

---

## Auth (public)

### POST /public/auth/login
```json
// Request
{ "email": "user@edm.co.mz", "password": "secret" }

// Response 200
{
  "token": "<jwt>",
  "user": { "id": 1, "name": "...", "email": "...", "role": "CA" }
}

// Response 401
{ "error": "invalid_credentials" }
```

### POST /public/auth/forgot-password
```json
// Request
{ "email": "user@edm.co.mz" }

// Response 200
{ "message": "reset email sent" }
```
Generates a `password_reset_token` (UUID), stores it + expiry (1h) in `users`, sends email.

### POST /public/auth/reset-password
```json
// Request
{ "token": "<reset-token>", "password": "newpassword" }

// Response 200
{ "message": "password updated" }

// Response 400
{ "error": "invalid_or_expired_token" }
```

---

## Users (private)

| Method | Path | Description | Roles |
|--------|------|-------------|-------|
| GET | /private/users | List all users | CA |
| POST | /private/users | Create user | CA |
| GET | /private/users/me | Current user profile | All |
| GET | /private/users/:id | Single user | CA |
| PUT | /private/users/:id | Update user | CA |
| DELETE | /private/users/:id | Soft delete | CA |

### POST /private/users — Body
```json
{
  "name": "João Silva",
  "email": "joao.silva@edm.co.mz",
  "password": "temp1234",
  "role": "DIRECAO"
}
```

---

## Organisation (private)

### Pelouros

| Method | Path | Roles |
|--------|------|-------|
| GET | /private/pelouros | CA, PELOURO |
| POST | /private/pelouros | CA |
| GET | /private/pelouros/:id | CA, PELOURO |
| PUT | /private/pelouros/:id | CA |
| DELETE | /private/pelouros/:id | CA |

**POST/PUT body:**
```json
{ "name": "Redução de Perdas", "description": "...", "responsible_id": 5 }
```

### Direções

| Method | Path | Roles |
|--------|------|-------|
| GET | /private/direcoes | CA, PELOURO, DIRECAO |
| POST | /private/direcoes | CA, PELOURO |
| GET | /private/direcoes/:id | CA, PELOURO, DIRECAO |
| PUT | /private/direcoes/:id | CA, PELOURO |
| DELETE | /private/direcoes/:id | CA |

**POST/PUT body:**
```json
{ "name": "Direcção Técnica", "pelouro_id": 1, "responsible_id": 7, "description": "..." }
```

### Departamentos

| Method | Path | Roles |
|--------|------|-------|
| GET | /private/departamentos | CA, PELOURO, DIRECAO, DEPARTAMENTO |
| POST | /private/departamentos | CA, PELOURO, DIRECAO |
| GET | /private/departamentos/:id | All |
| PUT | /private/departamentos/:id | CA, PELOURO, DIRECAO |
| DELETE | /private/departamentos/:id | CA |
| POST | /private/departamentos/:id/users | CA, DIRECAO — add user |
| DELETE | /private/departamentos/:id/users/:user_id | CA, DIRECAO — remove user |

### Organisation Tree

| Method | Path | Description |
|--------|------|-------------|
| GET | /private/org/tree | Full org hierarchy (CA → Pelouro → Direção → Departamento) |

**Response:**
```json
{
  "pelouros": [
    {
      "id": 1, "name": "...", "responsible": {...},
      "direcoes": [
        {
          "id": 2, "name": "...",
          "departamentos": [{ "id": 3, "name": "...", "users": [...] }]
        }
      ]
    }
  ]
}
```

---

## Geography (private)

### Regiões

| Method | Path | Roles |
|--------|------|-------|
| GET | /private/geo/regioes | All |
| POST | /private/geo/regioes | CA, DIRECAO |
| GET | /private/geo/regioes/:id | All |
| PUT | /private/geo/regioes/:id | CA, DIRECAO |
| DELETE | /private/geo/regioes/:id | CA |

**POST/PUT body:**
```json
{
  "name": "Norte",
  "code": "REG-N",
  "responsible_id": 3,
  "polygon": {
    "type": "Polygon",
    "coordinates": [[[32.5, -14.8], [35.0, -14.8], [35.0, -12.0], [32.5, -12.0], [32.5, -14.8]]]
  }
}
```

### ASCs

| Method | Path | Roles |
|--------|------|-------|
| GET | /private/geo/ascs | All |
| POST | /private/geo/ascs | CA, DIRECAO |
| GET | /private/geo/ascs/:id | All |
| PUT | /private/geo/ascs/:id | CA, DIRECAO |
| DELETE | /private/geo/ascs/:id | CA |

**POST/PUT body:**
```json
{
  "name": "ASC Pemba",
  "code": "ASC-PMB",
  "regiao_id": 1,
  "responsible_id": 10,
  "director_id": 11,
  "polygon": { "type": "Polygon", "coordinates": [[...]] }
}
```

---

## Projects (private)

| Method | Path | Description | Roles |
|--------|------|-------------|-------|
| GET | /private/projects | List (filtered by caller's visibility) | All |
| POST | /private/projects | Create project | All |
| GET | /private/projects/:id | Single + metadata | All |
| PUT | /private/projects/:id | Update | Owner role |
| DELETE | /private/projects/:id | Soft delete | Owner role or CA |
| GET | /private/projects/:id/tree | Full tree: project → tasks → milestones | All |

**GET /private/projects** — query params:
- `creator_type` — filter by CA | PELOURO | DIRECAO | DEPARTAMENTO
- `parent_id` — children of a specific project
- `status` — ACTIVE | COMPLETED | CANCELLED
- `page`, `limit`

**POST /private/projects body:**
```json
{
  "title": "Redução de Perdas",
  "description": "...",
  "creator_type": "CA",
  "creator_org_id": null,
  "parent_id": null,
  "weight": 100.0,
  "start_date": "2026-01-01",
  "end_date": "2026-12-31"
}
```

**On update:** notify parent org responsible users + all CA users via email + in-app notification.

---

## Tasks (private)

| Method | Path | Description |
|--------|------|-------------|
| GET | /private/tasks?project_id=X | List tasks for project |
| POST | /private/tasks?project_id=X | Create task |
| GET | /private/tasks/:id | Single task |
| PUT | /private/tasks/:id | Update task (triggers notifications) |
| DELETE | /private/tasks/:id | Soft delete |

**POST body:**
```json
{
  "title": "Efectuar 50.000 inspectores",
  "description": "...",
  "owner_type": "DIRECAO",
  "owner_id": 2,
  "frequency": "MONTHLY",
  "goal_label": "inspectores realizados",
  "start_value": 0,
  "target_value": 50000,
  "weight": 60.0,
  "start_date": "2026-01-01",
  "end_date": "2026-12-31",
  "parent_task_id": null,
  "scopes": [
    { "scope_type": "ASC", "scope_id": 5 },
    { "scope_type": "ASC", "scope_id": 6 }
  ]
}
```

**Notification on update:**
- If task owned by DIRECAO → notify PELOURO responsible + all CA users
- If task owned by DEPARTAMENTO → notify DIRECAO responsible → PELOURO → CA
- Also notify ASC director if task is scoped to their ASC

---

## Milestones (private)

| Method | Path | Description |
|--------|------|-------------|
| GET | /private/milestones?task_id=X | List milestones for task |
| POST | /private/milestones?task_id=X | Create milestone |
| GET | /private/milestones/:id | Single |
| PUT | /private/milestones/:id | Update (triggers task current_value recalc + notifications) |
| DELETE | /private/milestones/:id | Soft delete |
| POST | /private/milestones/:id/photo | Upload photo (multipart/form-data, max 5MB) |

**POST body:**
```json
{
  "title": "Semana 1 - Inspecções Pemba",
  "description": "...",
  "scope_type": "ASC",
  "scope_id": 5,
  "planned_value": 100,
  "planned_date": "2026-01-07",
  "notes": ""
}
```

**PUT body (update progress):**
```json
{
  "achieved_value": 87,
  "achieved_date": "2026-01-08",
  "status": "DONE",
  "notes": "Condições climatéricas adversas reduziram produção"
}
```

**On PUT:** recalculate `tasks.current_value` = SUM of achieved_value (non-blocked milestones).
Refresh `performance_cache` for affected entities.
Notify up the chain.

---

## Blockers / Impedimentos (private)

| Method | Path | Description |
|--------|------|-------------|
| GET | /private/blockers | List (filtered by entity or status) |
| POST | /private/blockers | Report a blocker |
| GET | /private/blockers/:id | Single |
| PUT | /private/blockers/:id/approve | Approve (superior) |
| PUT | /private/blockers/:id/reject | Reject with reason |

**POST body:**
```json
{
  "entity_type": "MILESTONE",
  "entity_id": 42,
  "blocker_type": "LOGISTIC",
  "description": "Veículos de transporte indisponíveis por avaria",
  "sla_days": 3
}
```

**PUT /approve body:** `{}`
**PUT /reject body:** `{ "rejection_reason": "Providenciar veículo alternativo" }`

**Auto-approval:** Background job runs hourly; sets `status = AUTO_APPROVED` where `auto_approve_at <= NOW() AND status = PENDING`.

---

## Dashboard & Analytics (private)

### GET /private/dashboard/summary
Overall company snapshot for current user's visibility scope.

**Response:**
```json
{
  "total_projects": 12,
  "total_tasks": 48,
  "milestones_done": 120,
  "milestones_pending": 35,
  "milestones_blocked": 4,
  "performance": {
    "execution_score": 72.4,
    "goal_score": 68.1,
    "total_score": 70.7,
    "traffic_light": "YELLOW"
  },
  "top_performers": [...],
  "alerts": [...]
}
```

---

### GET /private/dashboard/performance
Scores for a specific entity.

**Query params:**
- `entity_type` — CA | PELOURO | DIRECAO | DEPARTAMENTO | REGIAO | ASC | USER
- `entity_id` — corresponding ID (omit for CA)
- `period` — `2026-01` (month, optional; defaults to current month)

**Response:**
```json
{
  "entity_type": "DIRECAO",
  "entity_id": 2,
  "entity_name": "Direcção Técnica",
  "period": "2026-01",
  "execution_score": 75.0,
  "goal_score": 60.0,
  "total_score": 69.0,
  "traffic_light": "YELLOW",
  "tasks_total": 5,
  "tasks_completed": 2,
  "milestones_total": 30,
  "milestones_done": 22
}
```

---

### GET /private/dashboard/drill-down
Navigate the hierarchy: Country → Region → ASC → Direção → Departamento → User.

**Query params:**
- `level` — NATIONAL | REGIONAL | ASC | PELOURO | DIRECAO | DEPARTAMENTO
- `id` — entity id (omit for NATIONAL)
- `period` — `2026-01`

**Response:** array of children with their scores:
```json
{
  "level": "REGIONAL",
  "items": [
    {
      "id": 1, "name": "Norte", "type": "REGIAO",
      "execution_score": 80.0, "goal_score": 72.0, "total_score": 76.8,
      "traffic_light": "GREEN",
      "children_count": 4
    }
  ]
}
```

---

### GET /private/dashboard/map
GeoJSON FeatureCollection with performance data per region/ASC.

**Query params:**
- `level` — REGIONAL | ASC
- `period` — `2026-01`

**Response:** GeoJSON
```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": { "type": "Polygon", "coordinates": [[...]] },
      "properties": {
        "id": 1, "name": "Norte",
        "total_score": 76.8, "traffic_light": "GREEN"
      }
    }
  ]
}
```

---

### GET /private/dashboard/forecast
Linear velocity forecast for a task.

**Query params:** `task_id`

**Response:**
```json
{
  "task_id": 5,
  "title": "Efectuar 50.000 inspectores",
  "start_value": 0,
  "target_value": 50000,
  "current_value": 18000,
  "start_date": "2026-01-01",
  "end_date": "2026-12-31",
  "days_elapsed": 95,
  "days_remaining": 270,
  "velocity_per_day": 189.5,
  "projected_final_value": 69165,
  "will_reach_target": true,
  "alert": null
}
```
When `projected_final_value < target_value * 0.9`:
```json
{ "alert": "FORECAST_RISK", "message": "Ao ritmo actual, a tarefa irá atingir apenas 72% do objectivo." }
```

---

### GET /private/dashboard/top-performers
**Query params:**
- `entity_type` — ASC | DIRECAO | DEPARTAMENTO | USER
- `period` — `2026-01`
- `limit` — default 10

**Response:**
```json
{
  "period": "2026-01",
  "entity_type": "ASC",
  "ranking": [
    { "rank": 1, "id": 3, "name": "ASC Nacala", "total_score": 94.2, "traffic_light": "GREEN" },
    { "rank": 2, "id": 1, "name": "ASC Pemba",  "total_score": 88.7, "traffic_light": "GREEN" }
  ]
}
```

---

### GET /private/dashboard/timeline
Temporal trend chart data (improvement/degradation over time).

**Query params:**
- `entity_type`, `entity_id`
- `from` — `2025-01`
- `to` — `2026-04`

**Response:**
```json
{
  "entity_type": "DIRECAO",
  "entity_id": 2,
  "periods": [
    { "period": "2025-01", "total_score": 45.0, "traffic_light": "RED" },
    { "period": "2025-02", "total_score": 58.0, "traffic_light": "RED" },
    { "period": "2025-03", "total_score": 65.0, "traffic_light": "YELLOW" }
  ]
}
```

---

### GET /private/dashboard/distribution
Pie chart data — project/task distribution.

**Query params:**
- `entity_type`, `entity_id` (optional scope)
- `dimension` — BY_STATUS | BY_TRAFFIC_LIGHT | BY_OWNER_TYPE | BY_SCOPE

**Response:**
```json
{
  "dimension": "BY_TRAFFIC_LIGHT",
  "data": [
    { "label": "GREEN",  "count": 18, "percentage": 37.5 },
    { "label": "YELLOW", "count": 22, "percentage": 45.8 },
    { "label": "RED",    "count":  8, "percentage": 16.7 }
  ]
}
```

---

### GET /private/dashboard/benchmark
Compare two entities.

**Query params:** `entity_type`, `id_a`, `id_b`, `period`

**Response:**
```json
{
  "a": { "id": 1, "name": "ASC Pemba", "total_score": 88.7 },
  "b": { "id": 3, "name": "ASC Nacala", "total_score": 94.2 },
  "ratio": 1.06,
  "message": "ASC Nacala é 6% mais eficiente que ASC Pemba"
}
```

---

## Notifications (private)

| Method | Path | Description |
|--------|------|-------------|
| GET | /private/notifications | List for current user (unread first) |
| PUT | /private/notifications/:id/read | Mark as read |
| PUT | /private/notifications/read-all | Mark all as read |

**Query params for GET:** `is_read` (true/false), `type`, `page`, `limit`

---

## Audit (private)

| Method | Path | Roles |
|--------|------|-------|
| GET | /private/audit | CA, PELOURO, DIRECAO |

**Query params:** `entity_type`, `entity_id`, `from_date`, `to_date`, `page`, `limit`

**Response:**
```json
{
  "data": [
    {
      "id": 100,
      "entity_type": "milestone",
      "entity_id": 42,
      "action": "UPDATE",
      "changed_by": { "id": 7, "name": "João Silva" },
      "old_data": { "achieved_value": 0, "status": "PENDING" },
      "new_data":  { "achieved_value": 87, "status": "DONE" },
      "created_at": "2026-04-05T10:32:00Z"
    }
  ],
  "total": 1,
  "page": 1
}
```

---

## File Upload (photo on milestone)

`POST /private/milestones/:id/photo`
Content-Type: `multipart/form-data`
Field name: `photo`
Max size: **5 MB**
Accepted types: `image/jpeg`, `image/png`, `image/webp`

Storage: configurable via `STORAGE_BACKEND` env var:
- `local` → saved to `./uploads/milestones/`
- `s3` → uploaded to S3 bucket (requires `AWS_BUCKET`, `AWS_REGION`, `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`)

Returns: `{ "photo_url": "..." }`

---

## Pagination Convention

All list endpoints support:
- `page` — 0-based (default 0)
- `limit` — default 20, max 100

Response wrapper:
```json
{
  "data": [...],
  "total": 150,
  "page": 0,
  "limit": 20,
  "pages": 8
}
```

---

## Error Responses

```json
// 400
{ "error": "bad_request", "message": "field X is required" }

// 401
{ "error": "unauthorized", "message": "invalid or expired token" }

// 403
{ "error": "forbidden", "message": "insufficient role" }

// 404
{ "error": "not_found" }

// 409
{ "error": "conflict", "message": "email already in use" }

// 500
{ "error": "internal_error" }
```

---

## Role Permission Matrix

| Resource | CA | PELOURO | DIRECAO | DEPARTAMENTO |
|----------|----|---------|---------|--------------|
| Create Pelouro | ✓ | — | — | — |
| Create Direção | ✓ | ✓ | — | — |
| Create Departamento | ✓ | ✓ | ✓ | — |
| Create Project (own level) | ✓ | ✓ | ✓ | ✓ |
| Create Task (own scope) | ✓ | ✓ | ✓ | ✓ |
| Update Milestone | — | — | ✓ | ✓ |
| Approve Blocker | ✓ | ✓ | ✓ | — |
| View all dashboard | ✓ | Scoped | Scoped | Scoped |
| Manage Users | ✓ | — | — | — |
| View Audit | ✓ | ✓ | ✓ | — |
| Manage Geography | ✓ | — | ✓ | — |
