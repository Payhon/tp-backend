-- FEAT-0044: 电池列表新增生命周期-回退权限

DO $$
DECLARE
  battery_list_id varchar(36);
BEGIN
  SELECT id INTO battery_list_id
  FROM public.sys_ui_elements
  WHERE element_code = 'bms_battery_list'
  LIMIT 1;

  IF battery_list_id IS NULL THEN
    battery_list_id := 'f0f4c9b7-9e9c-4f3b-a3f0-1b8c2d6c7c10';
  END IF;

  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1010', battery_list_id,
    'bms_battery_list_action_lifecycle_rollback', 4, 13034,
    'bms_battery_list_action_lifecycle_rollback', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '生命周期-回退', NOW(), '页面元素权限', 'perm.bms_battery_list_action_lifecycle_rollback', ''
  WHERE NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_action_lifecycle_rollback'
  );

  UPDATE public.sys_ui_elements
  SET description = '生命周期-回退', multilingual = 'perm.bms_battery_list_action_lifecycle_rollback'
  WHERE element_code = 'bms_battery_list_action_lifecycle_rollback';
END $$;

WITH target_rows AS (
  SELECT tenant_id, org_type, COALESCE(ui_codes, '[]'::jsonb) AS ui_codes
  FROM public.org_type_permissions
  WHERE org_type IN ('PACK_FACTORY', 'DEALER', 'STORE')
    AND COALESCE(ui_codes, '[]'::jsonb) ? 'bms_battery_list'
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
          UNION ALL SELECT 'bms_battery_list_action_lifecycle_rollback'
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
