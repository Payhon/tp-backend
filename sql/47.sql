-- FEAT-0022: 移动端设备详情历史记录权限

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
      '9a2f86c4-1b45-4f85-9c4d-8b92b7d6c301',
      '0',
      'app_mobile_permissions',
      2,
      19990,
      '',
      'mdi:cellphone-cog',
      '',
      '["TENANT_ADMIN","SYS_ADMIN"]'::json,
      '移动端权限',
      NOW(),
      'FEAT-0022',
      'perm.app_mobile_permissions',
      ''
    );
  END IF;

  UPDATE public.sys_ui_elements
  SET
    parent_id = '0',
    element_type = 2,
    orders = 19990,
    param1 = '',
    param2 = 'mdi:cellphone-cog',
    param3 = '',
    authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    description = '移动端权限',
    remark = 'FEAT-0022',
    multilingual = 'perm.app_mobile_permissions',
    route_path = ''
  WHERE element_code = 'app_mobile_permissions';

  SELECT id INTO mobile_root_id
  FROM public.sys_ui_elements
  WHERE element_code = 'app_mobile_permissions'
  LIMIT 1;

  IF mobile_root_id IS NULL THEN
    mobile_root_id := '9a2f86c4-1b45-4f85-9c4d-8b92b7d6c301';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'app_device_detail_history'
  ) THEN
    INSERT INTO public.sys_ui_elements (
      id, parent_id, element_code, element_type, orders,
      param1, param2, param3, authority, description,
      created_at, remark, multilingual, route_path
    ) VALUES (
      '9a2f86c4-1b45-4f85-9c4d-8b92b7d6c302',
      mobile_root_id,
      'app_device_detail_history',
      4,
      19991,
      'app_device_detail_history',
      '',
      '1',
      '["TENANT_ADMIN","SYS_ADMIN"]'::json,
      '历史记录',
      NOW(),
      '页面元素权限',
      'perm.app_device_detail_history',
      ''
    );
  END IF;

  UPDATE public.sys_ui_elements
  SET
    parent_id = mobile_root_id,
    element_type = 4,
    orders = 19991,
    param1 = 'app_device_detail_history',
    param2 = '',
    param3 = '1',
    authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    description = '历史记录',
    remark = '页面元素权限',
    multilingual = 'perm.app_device_detail_history',
    route_path = ''
  WHERE element_code = 'app_device_detail_history';
END $$;
