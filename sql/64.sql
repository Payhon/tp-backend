-- FEAT-0069: 电池列表批量调拨权限

DO $$
DECLARE
	battery_list_id varchar(36);
BEGIN
	SELECT id INTO battery_list_id
	FROM public.sys_ui_elements
	WHERE element_code = 'bms_battery_list'
	LIMIT 1;

	IF battery_list_id IS NULL THEN
		RETURN;
	END IF;

	IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_batch_transfer') THEN
		INSERT INTO public.sys_ui_elements (
			id, parent_id, element_code, element_type, orders,
			param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
		) VALUES (
			'c63d2399-b4fb-4013-85d2-000cc6bba137', battery_list_id, 'bms_battery_list_batch_transfer', 4, 13029,
			'bms_battery_list_batch_transfer', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
			'批量调拨', NOW(), '页面元素权限', 'perm.bms_battery_list_batch_transfer', ''
		);
	END IF;

	UPDATE public.sys_ui_elements
	SET description = '批量调拨', multilingual = 'perm.bms_battery_list_batch_transfer'
	WHERE element_code = 'bms_battery_list_batch_transfer';
END $$;

WITH target_rows AS (
	SELECT tenant_id, org_type, COALESCE(ui_codes, '[]'::jsonb) AS ui_codes
	FROM public.org_type_permissions
	WHERE org_type IN ('BMS_FACTORY', 'PACK_FACTORY', 'DEALER')
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
					UNION ALL SELECT 'bms_battery_list_batch_transfer'
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
