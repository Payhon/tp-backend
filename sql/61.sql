-- FEAT-0063: PACK 厂小程序配置自助入口

DO $$
DECLARE
  bms_root_id varchar(36);
BEGIN
  SELECT id INTO bms_root_id
  FROM public.sys_ui_elements
  WHERE element_code = 'bms'
  LIMIT 1;

  IF bms_root_id IS NULL THEN
    bms_root_id := 'a753c525-780f-415f-a2b6-3d909c79f7f6';
  END IF;

  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1026', bms_root_id,
    'bms_pack_wxmp_config', 3, 1305,
    '/bms/pack-wxmp-config', 'mdi:wechat', 'self', '["TENANT_USER"]'::json,
    '小程序配置', NOW(), '', 'route.bms_pack_wxmp_config', 'view.bms_pack_wxmp_config'
  WHERE NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_pack_wxmp_config'
  );

  UPDATE public.sys_ui_elements
  SET
    parent_id = bms_root_id,
    element_type = 3,
    orders = 1305,
    param1 = '/bms/pack-wxmp-config',
    param2 = 'mdi:wechat',
    param3 = 'self',
    authority = '["TENANT_USER"]'::json,
    description = '小程序配置',
    multilingual = 'route.bms_pack_wxmp_config',
    route_path = 'view.bms_pack_wxmp_config'
  WHERE element_code = 'bms_pack_wxmp_config';
END $$;

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
          UNION ALL SELECT 'bms_pack_wxmp_config'
        ) AS raw_codes
        WHERE btrim(code) <> ''
          AND code <> 'bms_pack_factory'
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
