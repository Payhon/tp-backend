-- Version: 36
-- Description: add sys_ui_elements menu for dict management

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'management_dict'
  ) THEN
    INSERT INTO public.sys_ui_elements (
      id, parent_id, element_code, element_type, orders,
      param1, param2, param3, authority, description,
      created_at, remark, multilingual, route_path
    ) VALUES (
      '4e7e0b9e-6ee4-4b4b-9ef7-0b19f7d5f2f2',
      'e1ebd134-53df-3105-35f4-489fc674d173',
      'management_dict',
      3,
      45,
      '/management/dict',
      'mdi:book-open-page-variant',
      'self',
      '["SYS_ADMIN","TENANT_ADMIN"]'::json,
      '字典管理',
      now(),
      '',
      'route.management_dict',
      'view.management_dict'
    );
  END IF;
END $$;

