-- FEAT-0054: 后台附件管理菜单

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'management_attachment'
  ) THEN
    INSERT INTO public.sys_ui_elements (
      id, parent_id, element_code, element_type, orders,
      param1, param2, param3, authority, description,
      created_at, remark, multilingual, route_path
    ) VALUES (
      '7b4f2a31-4d6e-4a25-9c83-2d0f0a95c154',
      'e1ebd134-53df-3105-35f4-489fc674d173',
      'management_attachment',
      3,
      48,
      '/management/attachment',
      'mdi:paperclip',
      'self',
      '["SYS_ADMIN","TENANT_ADMIN"]'::json,
      '附件管理',
      now(),
      '',
      'route.management_attachment',
      'view.management_attachment'
    );
  END IF;

  UPDATE public.sys_ui_elements
  SET
    parent_id = 'e1ebd134-53df-3105-35f4-489fc674d173',
    element_type = 3,
    orders = 48,
    param1 = '/management/attachment',
    param2 = 'mdi:paperclip',
    param3 = 'self',
    authority = '["SYS_ADMIN","TENANT_ADMIN"]'::json,
    description = '附件管理',
    multilingual = 'route.management_attachment',
    route_path = 'view.management_attachment'
  WHERE element_code = 'management_attachment';
END $$;
