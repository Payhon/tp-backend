-- FEAT-0075: 移动端与 Web 共用的 BMS 保护状态整块显示权限；不自动授权。

DO $$
DECLARE
  mobile_root_id varchar(36);
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'app_mobile_permissions'
  ) THEN
    INSERT INTO public.sys_ui_elements (
      id, parent_id, element_code, element_type, orders,
      param1, param2, param3, authority, description,
      created_at, remark, multilingual, route_path
    ) VALUES (
      '9a2f86c4-1b45-4f85-9c4d-8b92b7d6c301', '0', 'app_mobile_permissions', 2, 19990,
      '', 'mdi:cellphone-cog', '', '["TENANT_ADMIN","SYS_ADMIN"]'::json, '移动端权限',
      NOW(), '移动端与 Web 共用显示权限', 'perm.app_mobile_permissions', ''
    );
  END IF;

  SELECT id INTO mobile_root_id
  FROM public.sys_ui_elements
  WHERE element_code = 'app_mobile_permissions'
  LIMIT 1;

  IF NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'app_device_detail_protection_status'
  ) THEN
    INSERT INTO public.sys_ui_elements (
      id, parent_id, element_code, element_type, orders,
      param1, param2, param3, authority, description,
      created_at, remark, multilingual, route_path
    ) VALUES (
      '9a2f86c4-1b45-4f85-9c4d-8b92b7d6c303', mobile_root_id, 'app_device_detail_protection_status', 4, 19992,
      'app_device_detail_protection_status', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json, 'BMS保护状态',
      NOW(), 'FEAT-0075：移动端与 Web 共用', 'perm.app_device_detail_protection_status', ''
    );
  END IF;

  UPDATE public.sys_ui_elements
  SET parent_id = mobile_root_id,
      element_type = 4,
      orders = 19992,
      param1 = 'app_device_detail_protection_status',
      param2 = '',
      param3 = '1',
      authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
      description = 'BMS保护状态',
      remark = 'FEAT-0075：移动端与 Web 共用',
      multilingual = 'perm.app_device_detail_protection_status',
      route_path = ''
  WHERE element_code = 'app_device_detail_protection_status';
END $$;
