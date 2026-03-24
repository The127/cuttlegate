-- Enable Row-Level Security for tenant isolation.
-- The application must SET LOCAL app.project_id = '<uuid>' at the start of
-- each transaction/request. RLS policies then restrict all reads and writes
-- to that project.
--
-- FORCE ROW LEVEL SECURITY ensures the table owner is also subject to RLS
-- (otherwise Postgres allows table owners to bypass RLS by default).

-- Flags
ALTER TABLE flags ENABLE ROW LEVEL SECURITY;
ALTER TABLE flags FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_flags ON flags
    USING (project_id = current_setting('app.project_id', true))
    WITH CHECK (project_id = current_setting('app.project_id', true));

-- Environments
ALTER TABLE environments ENABLE ROW LEVEL SECURITY;
ALTER TABLE environments FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_environments ON environments
    USING (project_id = current_setting('app.project_id', true))
    WITH CHECK (project_id = current_setting('app.project_id', true));

-- Flag environment states — no direct project_id column. Queries always go
-- through flags (which IS RLS-protected), so RLS is not needed here.
-- Same reasoning as rules below.

-- Project members
ALTER TABLE project_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_members FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_project_members ON project_members
    USING (project_id = current_setting('app.project_id', true))
    WITH CHECK (project_id = current_setting('app.project_id', true));

-- Segments
ALTER TABLE segments ENABLE ROW LEVEL SECURITY;
ALTER TABLE segments FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_segments ON segments
    USING (project_id = current_setting('app.project_id', true))
    WITH CHECK (project_id = current_setting('app.project_id', true));

-- API keys
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_api_keys ON api_keys
    USING (project_id = current_setting('app.project_id', true))
    WITH CHECK (project_id = current_setting('app.project_id', true));

-- Audit events
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_audit_events ON audit_events
    USING (project_id = current_setting('app.project_id', true))
    WITH CHECK (project_id = current_setting('app.project_id', true));

-- Evaluation events
ALTER TABLE evaluation_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE evaluation_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_eval_events ON evaluation_events
    USING (project_id = current_setting('app.project_id', true))
    WITH CHECK (project_id = current_setting('app.project_id', true));

-- Rules (scoped through flag, but have no direct project_id — skip for now,
-- rules are always queried via flag which is already RLS-protected).

-- Projects table itself is NOT RLS-protected — project listing is controlled
-- by the app layer (project_members join). Protecting it with RLS would
-- require a different GUC (user_id) and a more complex policy.
