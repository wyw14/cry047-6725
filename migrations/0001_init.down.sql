-- 0001_init.down.sql: rollback for the initial schema. Drop tables in reverse
-- dependency order. Idempotent: each DROP IF EXISTS is safe to re-run.

DROP TABLE IF EXISTS timeline_events;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS todos;
DROP TABLE IF EXISTS anomalies;
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS plan_versions;
DROP TABLE IF EXISTS maintenance_plans;
DROP TABLE IF EXISTS plan_templates;
DROP TABLE IF EXISTS facilities;
DROP TABLE IF EXISTS responsible_persons;
DROP TABLE IF EXISTS places;
