# Database Schema — KPI Platform

## Tech Notes
- PostgreSQL 15+ with **PostGIS** extension (for polygon/map support)
- GORM AutoMigrate handles standard tables; PostGIS extension and spatial indexes require manual `db.Exec` calls on startup
- All timestamps use `timestamptz` (UTC); soft-delete via `deleted_at`

---

## 1. users

```sql
id              bigserial     PRIMARY KEY
name            varchar(128)  NOT NULL
email           varchar(255)  NOT NULL UNIQUE
password        varchar(255)  NOT NULL           -- bcrypt hash
role            varchar(20)   NOT NULL           -- CA | PELOURO | DIRECAO | DEPARTAMENTO
active          boolean       DEFAULT true
password_reset_token    varchar(255)
password_reset_expires  timestamptz
last_login      timestamptz
created_at      timestamptz   NOT NULL
updated_at      timestamptz   NOT NULL
deleted_at      timestamptz
```

**Notes:**
- `role = CA` → user belongs to the board; multiple CA users exist (no org unit FK)
- `role = PELOURO/DIRECAO/DEPARTAMENTO` → link via their respective entity tables

---

## 2. pelouros (Portfolios / Board Divisions)

```sql
id              bigserial     PRIMARY KEY
name            varchar(255)  NOT NULL
description     text
responsible_id  bigint        REFERENCES users(id)
created_by      bigint        REFERENCES users(id)
created_at      timestamptz
updated_at      timestamptz
deleted_at      timestamptz
```

---

## 3. direcoes (Directorates)

```sql
id              bigserial     PRIMARY KEY
name            varchar(255)  NOT NULL
description     text
pelouro_id      bigint        NOT NULL REFERENCES pelouros(id)
responsible_id  bigint        REFERENCES users(id)
created_by      bigint        REFERENCES users(id)
created_at      timestamptz
updated_at      timestamptz
deleted_at      timestamptz
```

---

## 4. departamentos (Departments — lowest org level)

```sql
id              bigserial     PRIMARY KEY
name            varchar(255)  NOT NULL
description     text
direcao_id      bigint        NOT NULL REFERENCES direcoes(id)
responsible_id  bigint        REFERENCES users(id)
created_by      bigint        REFERENCES users(id)
created_at      timestamptz
updated_at      timestamptz
deleted_at      timestamptz
```

---

## 5. departamento_users (Many-to-Many)

```sql
user_id          bigint  NOT NULL REFERENCES users(id)
departamento_id  bigint  NOT NULL REFERENCES departamentos(id)
PRIMARY KEY (user_id, departamento_id)
```

---

## 6. regioes (Geographic Regions)

```sql
id              bigserial     PRIMARY KEY
name            varchar(255)  NOT NULL
code            varchar(50)
polygon         geometry(Polygon, 4326)   -- PostGIS; nullable while being set up
responsible_id  bigint        REFERENCES users(id)
created_at      timestamptz
updated_at      timestamptz
deleted_at      timestamptz
```

**Index:** `CREATE INDEX idx_regioes_polygon ON regioes USING GIST(polygon);`

---

## 7. ascs (Áreas de Serviço ao Cliente / Demographics)

```sql
id              bigserial     PRIMARY KEY
name            varchar(255)  NOT NULL
code            varchar(50)
regiao_id       bigint        REFERENCES regioes(id)
polygon         geometry(Polygon, 4326)
responsible_id  bigint        REFERENCES users(id)   -- 1 responsible user
director_id     bigint        REFERENCES users(id)   -- local director (receives emails for tasks in this ASC)
created_at      timestamptz
updated_at      timestamptz
deleted_at      timestamptz
```

**Index:** `CREATE INDEX idx_ascs_polygon ON ascs USING GIST(polygon);`

---

## 8. projects (Top-level Initiatives — any org level)

```sql
id              bigserial     PRIMARY KEY
title           varchar(500)  NOT NULL
description     text
creator_type    varchar(20)   NOT NULL   -- CA | PELOURO | DIRECAO | DEPARTAMENTO
creator_org_id  bigint                   -- pelouro_id / direcao_id / departamento_id; NULL for CA
parent_id       bigint        REFERENCES projects(id)   -- self-referencing hierarchy
weight          decimal(5,2)  DEFAULT 100.0             -- % weight in parent scoring
status          varchar(20)   DEFAULT 'ACTIVE'          -- ACTIVE | COMPLETED | CANCELLED | ARCHIVED
created_by      bigint        NOT NULL REFERENCES users(id)
start_date      date
end_date        date
created_at      timestamptz
updated_at      timestamptz
deleted_at      timestamptz
```

**Hierarchy example:**
```
CA Project (parent_id = NULL, creator_type = CA)
  └─ Pelouro Project (parent_id = CA.id, creator_type = PELOURO)
       └─ Direção Task-Project (parent_id = Pelouro.id, creator_type = DIRECAO)
```

---

## 9. tasks (Quantifiable Activities)

```sql
id              bigserial     PRIMARY KEY
project_id      bigint        NOT NULL REFERENCES projects(id)
parent_task_id  bigint        REFERENCES tasks(id)    -- sub-tasks at dept level
title           varchar(500)  NOT NULL
description     text
owner_type      varchar(20)   NOT NULL   -- DIRECAO | DEPARTAMENTO
owner_id        bigint        NOT NULL   -- direcao_id or departamento_id
frequency       varchar(20)   NOT NULL   -- DAILY | WEEKLY | MONTHLY | QUARTERLY | BIANNUAL | ANNUAL
goal_label      varchar(255)             -- e.g. "inspectores realizados", "perdas técnicas %"
start_value     decimal(15,2)            -- initial state (filled at task creation)
target_value    decimal(15,2) NOT NULL   -- desired end state
current_value   decimal(15,2) DEFAULT 0  -- auto-updated from milestones
weight          decimal(5,2)  DEFAULT 100.0
start_date      date
end_date        date
status          varchar(20)   DEFAULT 'ACTIVE'   -- ACTIVE | COMPLETED | CANCELLED
next_update_due date                             -- computed from frequency + last update
created_by      bigint        NOT NULL REFERENCES users(id)
created_at      timestamptz
updated_at      timestamptz
deleted_at      timestamptz
```

---

## 10. task_scopes (Geographic scope of a task)

```sql
id          bigserial    PRIMARY KEY
task_id     bigint       NOT NULL REFERENCES tasks(id)
scope_type  varchar(20)  NOT NULL   -- NACIONAL | REGIONAL | ASC
scope_id    bigint                  -- regiao_id or asc_id; NULL for NACIONAL
```

**Note:** A task can cover multiple scopes (e.g. Region A and Region B).
When a task is scoped to an ASC, the `ascs.director_id` user receives a notification email.

---

## 11. milestones

```sql
id              bigserial     PRIMARY KEY
task_id         bigint        NOT NULL REFERENCES tasks(id)
title           varchar(500)  NOT NULL
description     text
scope_type      varchar(20)              -- NACIONAL | REGIONAL | ASC (narrows task scope)
scope_id        bigint                   -- regiao_id or asc_id
planned_value   decimal(15,2) NOT NULL   -- what was planned for this milestone
achieved_value  decimal(15,2) DEFAULT 0  -- what was actually done
planned_date    date          NOT NULL
achieved_date   date
photo_url       varchar(1000)            -- single photo (stored in object storage / local)
status          varchar(20)   DEFAULT 'PENDING'   -- PENDING | IN_PROGRESS | DONE | BLOCKED
notes           text
created_by      bigint        NOT NULL REFERENCES users(id)
updated_by      bigint        REFERENCES users(id)
created_at      timestamptz
updated_at      timestamptz
deleted_at      timestamptz
```

**Trigger on update:** `current_value` on parent `tasks` is recalculated as `SUM(achieved_value)` of all non-blocked milestones.

---

## 12. blockers (Impedimentos)

```sql
id               bigserial    PRIMARY KEY
entity_type      varchar(20)  NOT NULL   -- TASK | MILESTONE
entity_id        bigint       NOT NULL
blocker_type     varchar(20)  NOT NULL   -- LOGISTIC | FINANCIAL | TECHNICAL | LEGAL
description      text         NOT NULL
reported_by      bigint       NOT NULL REFERENCES users(id)
approved_by      bigint       REFERENCES users(id)
status           varchar(20)  DEFAULT 'PENDING'   -- PENDING | APPROVED | REJECTED | AUTO_APPROVED
sla_days         int          DEFAULT 3
auto_approve_at  timestamptz                      -- computed: created_at + sla_days
resolved_at      timestamptz
rejection_reason text
created_at       timestamptz
updated_at       timestamptz
deleted_at       timestamptz
```

**Business rule:** If superior doesn't respond within `sla_days`, a background job sets `status = AUTO_APPROVED`.
**Performance rule:** When `status IN (APPROVED, AUTO_APPROVED)`, the associated milestone is excluded from execution score calculations.

---

## 13. audit_logs

```sql
id           bigserial    PRIMARY KEY
entity_type  varchar(50)  NOT NULL   -- project | task | milestone | blocker | user | ...
entity_id    bigint       NOT NULL
changed_by   bigint       NOT NULL REFERENCES users(id)
action       varchar(20)  NOT NULL   -- CREATE | UPDATE | DELETE
old_data     jsonb
new_data     jsonb
ip_address   varchar(45)
created_at   timestamptz  NOT NULL
```

**Index:** `CREATE INDEX idx_audit_entity ON audit_logs(entity_type, entity_id);`

---

## 14. notifications

```sql
id            bigserial    PRIMARY KEY
user_id       bigint       NOT NULL REFERENCES users(id)
title         varchar(255) NOT NULL
message       text         NOT NULL
type          varchar(40)  NOT NULL
              -- TASK_UPDATE | MILESTONE_UPDATE | BLOCKER_CREATED | BLOCKER_RESOLVED
              -- FORECAST_RISK | DELAY_ALERT | MILESTONE_OVERDUE | GOAL_AT_RISK
entity_type   varchar(30)
entity_id     bigint
is_read       boolean      DEFAULT false
email_sent    boolean      DEFAULT false
email_sent_at timestamptz
created_at    timestamptz  NOT NULL
```

---

## 15. performance_cache

Pre-computed scores, refreshed on every milestone update and nightly via cron.

```sql
id                  bigserial    PRIMARY KEY
entity_type         varchar(30)  NOT NULL
                    -- CA | PELOURO | DIRECAO | DEPARTAMENTO | REGIAO | ASC | USER
entity_id           bigint       NOT NULL   -- 0 for CA (singleton)
period              date         NOT NULL   -- first day of the month
execution_score     decimal(5,2)            -- % planned vs achieved milestones
goal_score          decimal(5,2)            -- % progress toward goal value
total_score         decimal(5,2)            -- (execution * 0.6) + (goal * 0.4)
traffic_light       varchar(10)             -- GREEN | YELLOW | RED
tasks_total         int          DEFAULT 0
tasks_completed     int          DEFAULT 0
milestones_total    int          DEFAULT 0
milestones_done     int          DEFAULT 0
milestones_blocked  int          DEFAULT 0
computed_at         timestamptz
UNIQUE (entity_type, entity_id, period)
```

---

## Scoring Formulas

```
Execution Score  = (SUM(achieved_value of non-blocked milestones) /
                    SUM(planned_value of non-blocked milestones)) * 100

Goal Score       = ((current_value - start_value) /
                    (target_value  - start_value)) * 100
                   capped at 100 if over-achieved

Performance Score = (Execution * 0.6) + (Goal * 0.4)

Traffic Light:
  GREEN  → score >= 90
  YELLOW → score >= 60 && score < 90
  RED    → score <  60
```

---

## Forecast Formula (linear velocity)

```
velocity              = (current_value - start_value) / days_elapsed
days_remaining        = end_date - today
projected_final_value = current_value + (velocity * days_remaining)

Will reach target?  projected_final_value >= target_value
Alert threshold:    if projected_final_value < (target_value * 0.9) → FORECAST_RISK notification
```

---

## AutoMigrate order (respects FK dependencies)

```go
// app/app.go — LoadDatabase()
models := []interface{}{
    &model.User{},
    &model.Pelouro{},
    &model.Direcao{},
    &model.Departamento{},
    &model.DepartamentoUser{},
    &model.Regiao{},
    &model.ASC{},
    &model.Project{},
    &model.Task{},
    &model.TaskScope{},
    &model.Milestone{},
    &model.Blocker{},
    &model.AuditLog{},
    &model.Notification{},
    &model.PerformanceCache{},
}
```

After AutoMigrate, run:
```go
dao.Database.Exec("CREATE EXTENSION IF NOT EXISTS postgis;")
dao.Database.Exec("CREATE INDEX IF NOT EXISTS idx_regioes_polygon ON regioes USING GIST(polygon);")
dao.Database.Exec("CREATE INDEX IF NOT EXISTS idx_ascs_polygon    ON ascs    USING GIST(polygon);")
dao.Database.Exec("CREATE INDEX IF NOT EXISTS idx_audit_entity    ON audit_logs(entity_type, entity_id);")
```
