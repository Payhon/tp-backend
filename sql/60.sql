-- FEAT-0062: 用户质保信息 + PACK 质保卡片开关

CREATE TABLE IF NOT EXISTS public.user_warranty_infos (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	user_id varchar(36) NOT NULL,
	contact_name varchar(100) NULL,
	contact_phone varchar(50) NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT user_warranty_infos_pkey PRIMARY KEY (id),
	CONSTRAINT user_warranty_infos_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE public.user_warranty_infos IS '终端用户质保联系人信息';
COMMENT ON COLUMN public.user_warranty_infos.tenant_id IS '租户ID';
COMMENT ON COLUMN public.user_warranty_infos.user_id IS '终端用户ID';
COMMENT ON COLUMN public.user_warranty_infos.contact_name IS '质保联系人姓名';
COMMENT ON COLUMN public.user_warranty_infos.contact_phone IS '质保联系人电话';

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_warranty_infos_tenant_user
	ON public.user_warranty_infos (tenant_id, user_id);

CREATE INDEX IF NOT EXISTS idx_user_warranty_infos_user
	ON public.user_warranty_infos (user_id);

ALTER TABLE public.device_batteries
	ADD COLUMN IF NOT EXISTS warranty_months int4 NULL,
	ADD COLUMN IF NOT EXISTS warranty_start_date timestamptz NULL,
	ADD COLUMN IF NOT EXISTS warranty_manual_override bool NOT NULL DEFAULT false,
	ADD COLUMN IF NOT EXISTS warranty_updated_at timestamptz NULL,
	ADD COLUMN IF NOT EXISTS warranty_updated_by varchar(36) NULL;

COMMENT ON COLUMN public.device_batteries.warranty_months IS '质保时长（月），优先来自 BMS 型号，可按电池覆盖';
COMMENT ON COLUMN public.device_batteries.warranty_start_date IS '质保开始日期，首次激活时写入';
COMMENT ON COLUMN public.device_batteries.warranty_manual_override IS '质保信息是否人工覆盖；为 true 时激活流程不再自动覆盖到期日';
COMMENT ON COLUMN public.device_batteries.warranty_updated_at IS '质保信息最后更新时间';
COMMENT ON COLUMN public.device_batteries.warranty_updated_by IS '质保信息最后更新人';

UPDATE public.device_batteries dbat
SET warranty_months = bm.warranty_months
FROM public.battery_bms_models bm
WHERE dbat.warranty_months IS NULL
  AND dbat.battery_model_id = bm.id
  AND bm.warranty_months IS NOT NULL;

UPDATE public.device_batteries
SET warranty_manual_override = true
WHERE warranty_expire_date IS NOT NULL
  AND warranty_manual_override = false;

ALTER TABLE public.pack_wxmp_configs
	ADD COLUMN IF NOT EXISTS warranty_cards_enabled bool NOT NULL DEFAULT true;

COMMENT ON COLUMN public.pack_wxmp_configs.warranty_cards_enabled IS '是否启用移动端质保页关联电池卡片';

DO $$
DECLARE
  battery_detail_id varchar(36);
BEGIN
  SELECT id INTO battery_detail_id
  FROM public.sys_ui_elements
  WHERE element_code = 'bms_battery_list_detail'
  LIMIT 1;

  IF battery_detail_id IS NULL THEN
    battery_detail_id := 'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1001';
  END IF;

  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1025', battery_detail_id,
    'bms_battery_detail_warranty', 4, 13042,
    'bms_battery_detail_warranty', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '质保', NOW(), '页面元素权限', 'perm.bms_battery_detail_warranty', ''
  WHERE NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_detail_warranty'
  );

  UPDATE public.sys_ui_elements
  SET description = '质保', multilingual = 'perm.bms_battery_detail_warranty'
  WHERE element_code = 'bms_battery_detail_warranty';
END $$;

WITH target_rows AS (
  SELECT tenant_id, org_type, COALESCE(ui_codes, '[]'::jsonb) AS ui_codes
  FROM public.org_type_permissions
  WHERE org_type IN ('PACK_FACTORY', 'DEALER')
    AND (
      COALESCE(ui_codes, '[]'::jsonb) ? 'bms_battery_list'
      OR COALESCE(ui_codes, '[]'::jsonb) ? 'bms_battery_list_detail'
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
          UNION ALL SELECT 'bms_battery_detail_warranty'
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
          UNION ALL SELECT 'bms_pack_factory'
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
