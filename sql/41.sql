-- FEAT-0012: 拆分 BMS 板型号与电池型号

-- 1) 历史表重命名：battery_models(旧) -> battery_bms_models
DO $$
BEGIN
    IF to_regclass('public.battery_bms_models') IS NULL
        AND to_regclass('public.battery_models') IS NOT NULL
        AND (
            EXISTS (
                SELECT 1
                FROM information_schema.columns
                WHERE table_schema = 'public'
                  AND table_name = 'battery_models'
                  AND column_name = 'device_config_id'
            )
            OR EXISTS (
                SELECT 1
                FROM information_schema.columns
                WHERE table_schema = 'public'
                  AND table_name = 'battery_models'
                  AND column_name = 'voltage_rated'
            )
        ) THEN
        ALTER TABLE public.battery_models RENAME TO battery_bms_models;
    END IF;
END $$;

-- 2) 确保 BMS 板型号表存在（租户管理员使用）
CREATE TABLE IF NOT EXISTS public.battery_bms_models (
    id varchar(36) PRIMARY KEY,
    name varchar(100) NOT NULL,
    device_config_id varchar(36),
    voltage_rated float,
    capacity_rated float,
    cell_count int,
    nominal_power float,
    warranty_months int,
    description text,
    tenant_id varchar(36) NOT NULL,
    created_at timestamptz DEFAULT NOW(),
    updated_at timestamptz DEFAULT NOW()
);

COMMENT ON TABLE public.battery_bms_models IS 'BMS板型号表（原 battery_models）';

-- 3) 新建电池型号表（机构维度）
CREATE TABLE IF NOT EXISTS public.battery_models (
    id varchar(36) PRIMARY KEY,
    seq_no smallint CHECK (seq_no BETWEEN 1 AND 255),
    name varchar(64) NOT NULL,
    org_id varchar(36),
    tenant_id varchar(36) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.battery_models IS '电池型号表（租户下机构维度）';
COMMENT ON COLUMN public.battery_models.seq_no IS '序号（1-255）';
COMMENT ON COLUMN public.battery_models.org_id IS '机构ID（当前登录用户所属机构）';

CREATE INDEX IF NOT EXISTS idx_battery_models_tenant_org_id
    ON public.battery_models (tenant_id, org_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_battery_models_tenant_org_seq_no
    ON public.battery_models (tenant_id, org_id, seq_no)
    WHERE org_id IS NOT NULL AND seq_no IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_battery_models_tenant_org_name
    ON public.battery_models (tenant_id, org_id, name)
    WHERE org_id IS NOT NULL;

-- 4) 迁移历史电池型号数据（保留原ID，避免 device_batteries.battery_model_id 失联）
INSERT INTO public.battery_models (id, seq_no, name, org_id, tenant_id, created_at, updated_at)
SELECT
    bm.id,
    NULL::smallint AS seq_no,
    bm.name,
    NULL::varchar(36) AS org_id,
    bm.tenant_id,
    COALESCE(bm.created_at, NOW()),
    COALESCE(bm.updated_at, NOW())
FROM public.battery_bms_models bm
WHERE NOT EXISTS (
    SELECT 1 FROM public.battery_models m WHERE m.id = bm.id
);

-- 5) 新增电芯品牌表（租户全局）
CREATE TABLE IF NOT EXISTS public.battery_cell_brands (
    id varchar(36) PRIMARY KEY,
    tenant_id varchar(36) NOT NULL,
    seq_no smallint NOT NULL CHECK (seq_no BETWEEN 1 AND 255),
    name varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.battery_cell_brands IS '电芯品牌表（租户全局）';
COMMENT ON COLUMN public.battery_cell_brands.seq_no IS '序号（1-255）';
COMMENT ON COLUMN public.battery_cell_brands.name IS '电芯品牌名（<=16字符）';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public' AND indexname = 'uq_battery_cell_brands_tenant_seq_no'
    ) THEN
        CREATE UNIQUE INDEX uq_battery_cell_brands_tenant_seq_no
            ON public.battery_cell_brands (tenant_id, seq_no);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public' AND indexname = 'uq_battery_cell_brands_tenant_name'
    ) THEN
        CREATE UNIQUE INDEX uq_battery_cell_brands_tenant_name
            ON public.battery_cell_brands (tenant_id, name);
    END IF;
END $$;

-- 6) 菜单与文案
DO $$
DECLARE
    bms_root_id varchar(36);
    bms_battery_id varchar(36);
BEGIN
    SELECT id INTO bms_root_id FROM public.sys_ui_elements WHERE element_code = 'bms' LIMIT 1;
    IF bms_root_id IS NULL THEN
        bms_root_id := 'a753c525-780f-415f-a2b6-3d909c79f7f6';
    END IF;

    SELECT id INTO bms_battery_id FROM public.sys_ui_elements WHERE element_code = 'bms_battery' LIMIT 1;
    IF bms_battery_id IS NULL THEN
        bms_battery_id := bms_root_id;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_cell_brand'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description,
            created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1011',
            bms_battery_id,
            'bms_battery_cell_brand',
            3,
            13014,
            '/bms/battery/cell-brand',
            'mdi:alpha-c-circle-outline',
            'self',
            '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '电芯品牌管理',
            NOW(),
            'FEAT-0012',
            'route.bms_battery_cell_brand',
            'view.bms_battery_cell_brand'
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_bms_model'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description,
            created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1014',
            bms_battery_id,
            'bms_battery_bms_model',
            3,
            13015,
            '/bms/battery/bms-model',
            'mdi:chip',
            'self',
            '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            'BMS型号管理',
            NOW(),
            'FEAT-0012',
            'route.bms_battery_bms_model',
            'view.bms_battery_bms_model'
        );
    END IF;

    UPDATE public.sys_ui_elements
    SET description = '新增BMS', multilingual = 'perm.bms_battery_list_add'
    WHERE element_code = 'bms_battery_list_add';
END $$;

-- 7) PACK 厂家权限补齐电池型号菜单（机构权限体系）
WITH target_rows AS (
    SELECT tenant_id, org_type, COALESCE(ui_codes, '[]'::jsonb) AS ui_codes
    FROM public.org_type_permissions
    WHERE org_type = 'PACK_FACTORY'
), merged_rows AS (
    SELECT
        tr.tenant_id,
        tr.org_type,
        (
            SELECT COALESCE(jsonb_agg(code ORDER BY code), '[]'::jsonb)
            FROM (
                SELECT DISTINCT code
                FROM (
                    SELECT jsonb_array_elements_text(tr.ui_codes) AS code
                    UNION ALL SELECT 'bms_battery_model'
                ) AS raw_codes
                WHERE btrim(code) <> ''
            ) AS dedup_codes
        ) AS merged_codes
    FROM target_rows tr
)
UPDATE public.org_type_permissions otp
SET
    ui_codes = mr.merged_codes,
    updated_at = NOW()
FROM merged_rows mr
WHERE otp.tenant_id = mr.tenant_id
  AND otp.org_type = mr.org_type;
