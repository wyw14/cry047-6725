-- 0001_init.up.sql: initial schema for the public-facility preventive
-- maintenance collaboration platform. Idempotent: every CREATE TABLE IF NOT
-- EXISTS / CREATE INDEX IF NOT EXISTS.

-- Places (场所).
CREATE TABLE IF NOT EXISTS places (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('library','exhibit','museum','community','other')),
    address     TEXT NOT NULL,
    code         TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version     INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_places_type ON places(type);
CREATE INDEX IF NOT EXISTS idx_places_created_at ON places(created_at);

-- Responsible persons (责任人).
CREATE TABLE IF NOT EXISTS responsible_persons (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    email        TEXT NOT NULL,
    phone        TEXT NOT NULL,
    department   TEXT NOT NULL DEFAULT '',
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version      INTEGER NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_responsible_persons_email ON responsible_persons(email) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_responsible_persons_department ON responsible_persons(department);

-- Facilities (设施).
CREATE TABLE IF NOT EXISTS facilities (
    id                     TEXT PRIMARY KEY,
    place_id               TEXT NOT NULL REFERENCES places(id) ON DELETE RESTRICT,
    name                   TEXT NOT NULL,
    code                   TEXT NOT NULL,
    category               TEXT NOT NULL,
    responsible_person_id  TEXT NOT NULL REFERENCES responsible_persons(id) ON DELETE RESTRICT,
    criticality            TEXT NOT NULL CHECK (criticality IN ('critical','important','standard')),
    status                 TEXT NOT NULL CHECK (status IN ('normal','pending_maintenance','overdue','restricted_use','under_repair','recovered')),
    description            TEXT NOT NULL DEFAULT '',
    alternative_facility_id TEXT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version                INTEGER NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_facilities_code_place ON facilities(code, place_id);
CREATE INDEX IF NOT EXISTS idx_facilities_place ON facilities(place_id);
CREATE INDEX IF NOT EXISTS idx_facilities_responsible ON facilities(responsible_person_id);
CREATE INDEX IF NOT EXISTS idx_facilities_criticality ON facilities(criticality);
CREATE INDEX IF NOT EXISTS idx_facilities_status ON facilities(status);

-- Plan templates (保养计划模板).
CREATE TABLE IF NOT EXISTS plan_templates (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    cycle_days        INTEGER NOT NULL CHECK (cycle_days BETWEEN 1 AND 365),
    inspection_items  JSONB NOT NULL DEFAULT '[]'::JSONB,
    consumables       JSONB NOT NULL DEFAULT '[]'::JSONB,
    requires_shutdown BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version           INTEGER NOT NULL DEFAULT 1
);

-- Maintenance plans (保养计划).
CREATE TABLE IF NOT EXISTS maintenance_plans (
    id                 TEXT PRIMARY KEY,
    facility_id        TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    template_id        TEXT NOT NULL REFERENCES plan_templates(id) ON DELETE RESTRICT,
    current_cycle_days INTEGER NOT NULL CHECK (current_cycle_days BETWEEN 1 AND 365),
    next_due_date      TIMESTAMPTZ,
    last_executed_at   TIMESTAMPTZ,
    last_execution_id  TEXT,
    active             BOOLEAN NOT NULL DEFAULT TRUE,
    skipped_count      INTEGER NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version            INTEGER NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_maintenance_plans_facility_active
    ON maintenance_plans(facility_id) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_maintenance_plans_next_due ON maintenance_plans(next_due_date) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_maintenance_plans_template ON maintenance_plans(template_id);

-- Plan versions (计划版本).
CREATE TABLE IF NOT EXISTS plan_versions (
    id             TEXT PRIMARY KEY,
    plan_id        TEXT NOT NULL REFERENCES maintenance_plans(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    cycle_days     INTEGER NOT NULL,
    template_id    TEXT,
    changed_by     TEXT NOT NULL,
    changed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_plan_versions_plan ON plan_versions(plan_id, version_number);

-- Executions (执行表单).
CREATE TABLE IF NOT EXISTS executions (
    id                   TEXT PRIMARY KEY,
    plan_id              TEXT NOT NULL REFERENCES maintenance_plans(id) ON DELETE RESTRICT,
    facility_id          TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    executed_by          TEXT NOT NULL,
    executed_at          TIMESTAMPTZ NOT NULL,
    inspection_values    JSONB NOT NULL DEFAULT '[]'::JSONB,
    photos               JSONB NOT NULL DEFAULT '[]'::JSONB,
    consumables_consumed JSONB NOT NULL DEFAULT '[]'::JSONB,
    review_comment       TEXT NOT NULL DEFAULT '',
    reviewed_by          TEXT,
    reviewed_at          TIMESTAMPTZ,
    status               TEXT NOT NULL CHECK (status IN ('draft','submitted','reviewed','rejected')),
    idempotency_key      TEXT NOT NULL UNIQUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version              INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_executions_plan ON executions(plan_id);
CREATE INDEX IF NOT EXISTS idx_executions_facility ON executions(facility_id);
CREATE INDEX IF NOT EXISTS idx_executions_executed_at ON executions(executed_at);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);

-- Anomalies (异常).
CREATE TABLE IF NOT EXISTS anomalies (
    id                    TEXT PRIMARY KEY,
    facility_id           TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    execution_id          TEXT REFERENCES executions(id) ON DELETE SET NULL,
    discovered_by         TEXT NOT NULL,
    discovered_at         TIMESTAMPTZ NOT NULL,
    description           TEXT NOT NULL,
    severity              TEXT NOT NULL CHECK (severity IN ('low','medium','high','critical')),
    status                TEXT NOT NULL CHECK (status IN ('open','rectifying','reinspecting','recovered','closed_no_action')),
    rectification_measure TEXT NOT NULL DEFAULT '',
    rectified_by          TEXT,
    rectified_at          TIMESTAMPTZ,
    reinspection_result  TEXT NOT NULL DEFAULT '',
    reinspected_by       TEXT,
    reinspected_at       TIMESTAMPTZ,
    recovered_by         TEXT,
    recovered_at         TIMESTAMPTZ,
    idempotency_key       TEXT NOT NULL UNIQUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version               INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_anomalies_facility ON anomalies(facility_id);
CREATE INDEX IF NOT EXISTS idx_anomalies_status ON anomalies(status);
CREATE INDEX IF NOT EXISTS idx_anomalies_severity ON anomalies(severity);
CREATE INDEX IF NOT EXISTS idx_anomalies_discovered_at ON anomalies(discovered_at);

-- Todos (待办).
CREATE TABLE IF NOT EXISTS todos (
    id              TEXT PRIMARY KEY,
    facility_id     TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    plan_id         TEXT,
    anomaly_id      TEXT,
    assigned_to     TEXT NOT NULL,
    type            TEXT NOT NULL CHECK (type IN ('maintenance','rectification','reinspection','recovery','review')),
    due_date        TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('open','assigned','completed','skipped','overdue')),
    title           TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    version         INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_todos_assigned ON todos(assigned_to);
CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date);
CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);
CREATE INDEX IF NOT EXISTS idx_todos_facility ON todos(facility_id);

-- Notifications (本地通知中心).
CREATE TABLE IF NOT EXISTS notifications (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('maintenance_due','maintenance_overdue','anomaly_discovered','anomaly_recovered','plan_updated','assigned')),
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(user_id) WHERE read_at IS NULL;

-- Audit logs (审计日志).
CREATE TABLE IF NOT EXISTS audit_logs (
    id           TEXT PRIMARY KEY,
    action       TEXT NOT NULL CHECK (action IN ('create','update','delete','state_change','import','export','recover','skip','regenerate')),
    entity_type  TEXT NOT NULL,
    entity_id    TEXT NOT NULL,
    actor_id     TEXT NOT NULL,
    actor_name   TEXT NOT NULL,
    before_state JSONB,
    after_state  JSONB,
    request_id   TEXT NOT NULL DEFAULT '',
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Timeline events (设施时间线).
CREATE TABLE IF NOT EXISTS timeline_events (
    id          TEXT PRIMARY KEY,
    facility_id TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_type  TEXT NOT NULL,
    title       TEXT NOT NULL,
    actor       TEXT NOT NULL,
    detail      JSONB
);

CREATE INDEX IF NOT EXISTS idx_timeline_facility ON timeline_events(facility_id, occurred_at DESC);
