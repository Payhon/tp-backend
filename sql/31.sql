-- Version: 31
-- Description: Battery model link device_config + battery operation logs + battery import jobs

-- 1) battery model link device_configs
-- 兼容两种结构：
-- - 旧结构：battery_models（后续会被 41.sql 重命名为 battery_bms_models）
-- - 新结构：battery_bms_models（1.sql 已拆分）
DO $$
DECLARE
    target_table text;
BEGIN
    IF to_regclass('public.battery_bms_models') IS NOT NULL THEN
        target_table := 'battery_bms_models';
    ELSIF to_regclass('public.battery_models') IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'battery_models'
              AND column_name = 'voltage_rated'
        ) THEN
        -- 仅对旧版 battery_models（BMS 板型号）补充字段，避免污染新电池型号表
        target_table := 'battery_models';
    END IF;

    IF target_table IS NULL THEN
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = target_table
          AND column_name = 'device_config_id'
    ) THEN
        EXECUTE format('ALTER TABLE public.%I ADD COLUMN device_config_id varchar(36) NULL', target_table);
    END IF;

    EXECUTE format(
        'COMMENT ON COLUMN public.%I.device_config_id IS %L',
        target_table,
        '关联设备模板ID(device_configs.id)'
    );

    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_battery_models_device_config_id ON public.%I(device_config_id)', target_table);
END $$;

DO $$
DECLARE
    target_table text;
BEGIN
    IF to_regclass('public.battery_bms_models') IS NOT NULL THEN
        target_table := 'battery_bms_models';
    ELSIF to_regclass('public.battery_models') IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'battery_models'
              AND column_name = 'voltage_rated'
        ) THEN
        target_table := 'battery_models';
    END IF;

    IF target_table IS NULL THEN
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema = 'public'
          AND table_name = target_table
          AND constraint_name = 'fk_battery_models_device_config_id'
    ) THEN
        EXECUTE format(
            'ALTER TABLE public.%I ADD CONSTRAINT fk_battery_models_device_config_id FOREIGN KEY (device_config_id) REFERENCES public.device_configs(id) ON DELETE RESTRICT',
            target_table
        );
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
