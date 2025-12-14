-- BMS System Management menu seed
-- Version: 16
-- Description: Add BMS system management menus (account/role)

-- BMS 系统管理（目录）
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  '9e6a0c3a-2b7f-4d8c-9f1a-2d3b4c5d6e70',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_system',
  2,
  1310,
  '/bms/system',
  'mdi:cog',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '系统管理',
  NOW(),
  '',
  'route.bms_system',
  'layout.base'
);

-- 账号管理
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  'a61b4a30-9d22-4b11-9f11-1a2b3c4d5e61',
  '9e6a0c3a-2b7f-4d8c-9f1a-2d3b4c5d6e70',
  'bms_system_user',
  3,
  1311,
  '/bms/system/user',
  'mdi:account-cog',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '账号管理',
  NOW(),
  '',
  'route.bms_system_user',
  'view.bms_system_user'
);

-- 角色管理
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  'b72c5b41-8e33-4c22-8f22-2b3c4d5e6f72',
  '9e6a0c3a-2b7f-4d8c-9f1a-2d3b4c5d6e70',
  'bms_system_role',
  3,
  1312,
  '/bms/system/role',
  'mdi:account-key',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '角色管理',
  NOW(),
  '',
  'route.bms_system_role',
  'view.bms_system_role'
);

