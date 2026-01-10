-- Version: 31
-- Description: Battery model link device_config + battery operation logs + battery import jobs

-- 1) battery_models link device_configs
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'battery_models' AND column_name = 'device_config_id'
    ) THEN
        ALTER TABLE public.battery_models
            ADD COLUMN device_config_id varchar(36) NULL;
        COMMENT ON COLUMN public.battery_models.device_config_id IS '关联设备模板ID(device_configs.id)';

        CREATE INDEX IF NOT EXISTS idx_battery_models_device_config_id ON public.battery_models(device_config_id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'battery_models' AND constraint_name = 'fk_battery_models_device_config_id'
    ) THEN
        ALTER TABLE public.battery_models
            ADD CONSTRAINT fk_battery_models_device_config_id
            FOREIGN KEY (device_config_id) REFERENCES public.device_configs(id) ON DELETE RESTRICT;
    END IF;
END $$;

-- 2) battery operation logs (运营日志：围绕电池的业务操作)
CREATE TABLE IF NOT EXISTS public.battery_operation_logs (
    id BIGSERIAL PRIMARY KEY,
    tenant_id varchar(36) NOT NULL,
    device_id varchar(36) NOT NULL,
    device_number varchar(64) NOT NULL,
    operation_type varchar(32) NOT NULL, -- CREATE/IMPORT/TRANSFER/WARRANTY_SUBMIT/WARRANTY_HANDLE/MAINTENANCE_SUBMIT/MAINTENANCE_HANDLE/...
    operator_id varchar(36) NULL,
    occurred_at timestamptz(6) NOT NULL DEFAULT NOW(),
    description text NULL,
    extra jsonb NULL DEFAULT '{}'::jsonb
);

COMMENT ON TABLE public.battery_operation_logs IS '电池运营日志（围绕单个电池的业务操作记录）';
COMMENT ON COLUMN public.battery_operation_logs.device_number IS '电池编号/序列号（对应 devices.device_number / device_batteries.item_uuid）';
COMMENT ON COLUMN public.battery_operation_logs.operation_type IS '操作类型';
COMMENT ON COLUMN public.battery_operation_logs.operator_id IS '操作人（users.id）';
COMMENT ON COLUMN public.battery_operation_logs.occurred_at IS '操作时间';

CREATE INDEX IF NOT EXISTS idx_battery_operation_logs_tenant_time ON public.battery_operation_logs(tenant_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_battery_operation_logs_tenant_device_number ON public.battery_operation_logs(tenant_id, device_number);
CREATE INDEX IF NOT EXISTS idx_battery_operation_logs_tenant_device_id ON public.battery_operation_logs(tenant_id, device_id);

-- 3) battery import jobs (for progress + logs)
CREATE TABLE IF NOT EXISTS public.battery_import_jobs (
    id varchar(36) PRIMARY KEY,
    tenant_id varchar(36) NOT NULL,
    operator_id varchar(36) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'PENDING', -- PENDING/RUNNING/SUCCESS/FAILED
    total_rows int NOT NULL DEFAULT 0,
    processed_rows int NOT NULL DEFAULT 0,
    success_rows int NOT NULL DEFAULT 0,
    failed_rows int NOT NULL DEFAULT 0,
    error_message text NULL,
    started_at timestamptz(6) NULL,
    finished_at timestamptz(6) NULL,
    created_at timestamptz(6) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.battery_import_jobs IS '电池导入任务（用于前端显示进度/日志）';

CREATE INDEX IF NOT EXISTS idx_battery_import_jobs_tenant_created_at ON public.battery_import_jobs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_battery_import_jobs_tenant_status ON public.battery_import_jobs(tenant_id, status);

CREATE TABLE IF NOT EXISTS public.battery_import_job_logs (
    id BIGSERIAL PRIMARY KEY,
    job_id varchar(36) NOT NULL,
    tenant_id varchar(36) NOT NULL,
    row_number int NULL,
    level varchar(8) NOT NULL DEFAULT 'INFO', -- INFO/WARN/ERROR
    device_number varchar(64) NULL,
    message text NOT NULL,
    created_at timestamptz(6) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.battery_import_job_logs IS '电池导入任务日志（按行记录）';

CREATE INDEX IF NOT EXISTS idx_battery_import_job_logs_job_id ON public.battery_import_job_logs(job_id, id);
CREATE INDEX IF NOT EXISTS idx_battery_import_job_logs_tenant_created_at ON public.battery_import_job_logs(tenant_id, created_at DESC);
