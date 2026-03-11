-- FEAT-0012: BMS 历史数据查询 + 异步导出通知

-- 1) 属性历史明细表（append-only）
CREATE TABLE IF NOT EXISTS public.attribute_history_datas (
    id BIGSERIAL PRIMARY KEY,
    device_id varchar(36) NOT NULL,
    "key" varchar(255) NOT NULL,
    ts timestamptz(6) NOT NULL,
    bool_v bool NULL,
    number_v float8 NULL,
    string_v text NULL,
    tenant_id varchar(36) NOT NULL,
    created_at timestamptz(6) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.attribute_history_datas IS '属性历史明细（append-only，不回填历史）';
COMMENT ON COLUMN public.attribute_history_datas.device_id IS '设备ID';
COMMENT ON COLUMN public.attribute_history_datas.key IS '数据标识符';
COMMENT ON COLUMN public.attribute_history_datas.ts IS '上报时间';

CREATE INDEX IF NOT EXISTS idx_attr_history_tenant_device_ts
  ON public.attribute_history_datas (tenant_id, device_id, ts DESC);

CREATE INDEX IF NOT EXISTS idx_attr_history_tenant_device_key_ts
  ON public.attribute_history_datas (tenant_id, device_id, "key", ts DESC);

-- 2) BMS 历史导出任务表
CREATE TABLE IF NOT EXISTS public.bms_history_export_jobs (
    id varchar(36) PRIMARY KEY,
    tenant_id varchar(36) NOT NULL,
    creator_user_id varchar(36) NOT NULL,
    creator_org_id varchar(36) NULL,
    device_id varchar(36) NOT NULL,
    view_mode varchar(16) NOT NULL,
    start_time_ms bigint NOT NULL,
    end_time_ms bigint NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'PENDING', -- PENDING/RUNNING/SUCCESS/FAILED
    file_path varchar(1024) NULL,
    file_name varchar(255) NULL,
    file_size bigint NULL,
    file_expire_at timestamptz(6) NULL,
    error_message text NULL,
    downloaded_at timestamptz(6) NULL,
    started_at timestamptz(6) NULL,
    finished_at timestamptz(6) NULL,
    created_at timestamptz(6) NOT NULL DEFAULT NOW(),
    updated_at timestamptz(6) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.bms_history_export_jobs IS 'BMS 历史数据异步导出任务';
COMMENT ON COLUMN public.bms_history_export_jobs.view_mode IS '导出视图 long|wide';
COMMENT ON COLUMN public.bms_history_export_jobs.downloaded_at IS '下载完成时间（成功下载后写入）';
COMMENT ON COLUMN public.bms_history_export_jobs.file_expire_at IS '导出文件过期时间';

CREATE INDEX IF NOT EXISTS idx_bms_history_export_jobs_tenant_user_status
  ON public.bms_history_export_jobs (tenant_id, creator_user_id, status, downloaded_at, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_bms_history_export_jobs_tenant_created
  ON public.bms_history_export_jobs (tenant_id, created_at DESC);

-- 3) 菜单 + 按钮权限
DO $$
DECLARE
  bms_root_id varchar(36);
  bms_battery_id varchar(36);
  bms_history_id varchar(36);
BEGIN
  SELECT id INTO bms_root_id
  FROM public.sys_ui_elements
  WHERE element_code = 'bms'
  LIMIT 1;

  IF bms_root_id IS NULL THEN
    bms_root_id := 'a753c525-780f-415f-a2b6-3d909c79f7f6';
  END IF;

  SELECT id INTO bms_battery_id
  FROM public.sys_ui_elements
  WHERE element_code = 'bms_battery'
  LIMIT 1;

  IF bms_battery_id IS NULL THEN
    bms_battery_id := bms_root_id;
  END IF;

  -- 历史数据菜单
  IF NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_history'
  ) THEN
    INSERT INTO public.sys_ui_elements (
      id, parent_id, element_code, element_type, orders,
      param1, param2, param3, authority, description,
      created_at, remark, multilingual, route_path
    ) VALUES (
      'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1012',
      bms_battery_id,
      'bms_battery_history',
      3,
      13016,
      '/bms/battery/history',
      'mdi:chart-timeline-variant',
      'self',
      '["TENANT_ADMIN","SYS_ADMIN"]'::json,
      '历史数据',
      NOW(),
      'FEAT-0012',
      'route.bms_battery_history',
      'view.bms_battery_history'
    );
  END IF;

  UPDATE public.sys_ui_elements
  SET
    parent_id = bms_battery_id,
    element_type = 3,
    orders = 13016,
    param1 = '/bms/battery/history',
    param2 = 'mdi:chart-timeline-variant',
    param3 = 'self',
    authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    description = '历史数据',
    multilingual = 'route.bms_battery_history',
    route_path = 'view.bms_battery_history'
  WHERE element_code = 'bms_battery_history';

  SELECT id INTO bms_history_id
  FROM public.sys_ui_elements
  WHERE element_code = 'bms_battery_history'
  LIMIT 1;

  IF bms_history_id IS NULL THEN
    bms_history_id := 'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1012';
  END IF;

  -- 导出按钮权限
  IF NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_history_export'
  ) THEN
    INSERT INTO public.sys_ui_elements (
      id, parent_id, element_code, element_type, orders,
      param1, param2, param3, authority, description,
      created_at, remark, multilingual, route_path
    ) VALUES (
      'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1013',
      bms_history_id,
      'bms_battery_history_export',
      4,
      13061,
      'bms_battery_history_export',
      '',
      '1',
      '["TENANT_ADMIN","SYS_ADMIN"]'::json,
      '导出历史数据',
      NOW(),
      '页面元素权限',
      'perm.bms_battery_history_export',
      ''
    );
  END IF;

  UPDATE public.sys_ui_elements
  SET
    parent_id = bms_history_id,
    element_type = 4,
    orders = 13061,
    param1 = 'bms_battery_history_export',
    param2 = '',
    param3 = '1',
    authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    description = '导出历史数据',
    multilingual = 'perm.bms_battery_history_export',
    route_path = ''
  WHERE element_code = 'bms_battery_history_export';
END $$;

-- 4) 机构类型权限补齐（已具备电池管理权限的租户）
WITH target_rows AS (
  SELECT tenant_id, org_type, COALESCE(ui_codes, '[]'::jsonb) AS ui_codes
  FROM public.org_type_permissions
  WHERE org_type IN ('PACK_FACTORY', 'DEALER', 'STORE')
    AND (
      COALESCE(ui_codes, '[]'::jsonb) ? 'bms_battery'
      OR COALESCE(ui_codes, '[]'::jsonb) ? 'bms_battery_list'
    )
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
          UNION ALL SELECT 'bms_battery_history'
          UNION ALL SELECT 'bms_battery_history_export'
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
