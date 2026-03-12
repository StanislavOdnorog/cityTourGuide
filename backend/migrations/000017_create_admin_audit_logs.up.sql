CREATE TABLE admin_audit_logs (
    id         BIGSERIAL    PRIMARY KEY,
    actor_id   TEXT         NOT NULL DEFAULT '',
    action     TEXT         NOT NULL,
    resource_type TEXT      NOT NULL,
    resource_id   TEXT      NOT NULL DEFAULT '',
    http_method   TEXT      NOT NULL,
    request_path  TEXT      NOT NULL,
    trace_id      TEXT      NOT NULL DEFAULT '',
    payload       JSONB,
    status        TEXT      NOT NULL DEFAULT 'success',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_logs_actor_id ON admin_audit_logs (actor_id);
CREATE INDEX idx_admin_audit_logs_resource ON admin_audit_logs (resource_type, resource_id);
CREATE INDEX idx_admin_audit_logs_created_at ON admin_audit_logs (created_at);
