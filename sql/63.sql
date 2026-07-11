-- FEAT-0067: BMS 电池质保截止日期补偿任务

CREATE TABLE IF NOT EXISTS public.battery_warranty_recalc_jobs (
    id varchar(36) PRIMARY KEY,
    tenant_id varchar(36) NOT NULL,
    operator_id varchar(36) NULL,
    source varchar(32) NOT NULL, -- MODEL_CHANGE / MANUAL_SCAN
    scope_model_id varchar(36) NULL,
    status varchar(16) NOT NULL DEFAULT 'PENDING', -- PENDING/RUNNING/SUCCESS/FAILED
    total_rows int NOT NULL DEFAULT 0,
    processed_rows int NOT NULL DEFAULT 0,
    success_rows int NOT NULL DEFAULT 0,
    skipped_rows int NOT NULL DEFAULT 0,
    failed_rows int NOT NULL DEFAULT 0,
    error_message text NULL,
    started_at timestamptz(6) NULL,
    finished_at timestamptz(6) NULL,
    created_at timestamptz(6) NOT NULL DEFAULT NOW(),
    updated_at timestamptz(6) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.battery_warranty_recalc_jobs IS '电池质保截止日期补偿任务';
COMMENT ON COLUMN public.battery_warranty_recalc_jobs.source IS '任务来源：MODEL_CHANGE 型号质保时长变更；MANUAL_SCAN 后台手动扫描空截止日期';
COMMENT ON COLUMN public.battery_warranty_recalc_jobs.scope_model_id IS '型号变更任务的 BMS 型号 ID；手动扫描为空';

CREATE INDEX IF NOT EXISTS idx_battery_warranty_recalc_jobs_tenant_created_at
    ON public.battery_warranty_recalc_jobs(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_battery_warranty_recalc_jobs_tenant_status
    ON public.battery_warranty_recalc_jobs(tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_battery_warranty_recalc_jobs_scope_model
    ON public.battery_warranty_recalc_jobs(tenant_id, scope_model_id);

CREATE TABLE IF NOT EXISTS public.battery_warranty_recalc_job_logs (
    id BIGSERIAL PRIMARY KEY,
    job_id varchar(36) NOT NULL,
    tenant_id varchar(36) NOT NULL,
    level varchar(8) NOT NULL DEFAULT 'INFO', -- INFO/WARN/ERROR
    device_id varchar(36) NULL,
    device_number varchar(64) NULL,
    battery_model_id varchar(36) NULL,
    message text NOT NULL,
    created_at timestamptz(6) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.battery_warranty_recalc_job_logs IS '电池质保截止日期补偿任务日志';

CREATE INDEX IF NOT EXISTS idx_battery_warranty_recalc_job_logs_job_id
    ON public.battery_warranty_recalc_job_logs(job_id, id);

CREATE INDEX IF NOT EXISTS idx_battery_warranty_recalc_job_logs_tenant_created_at
    ON public.battery_warranty_recalc_job_logs(tenant_id, created_at DESC);
