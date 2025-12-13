-- BMS UI Menu Registration
-- Version: 14
-- Description: Register BMS related menus into sys_ui_elements for Web routes

-- Top level BMS menu
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  'a753c525-780f-415f-a2b6-3d909c79f7f6', -- BMS root
  '0',
  'bms',
  1,                -- menu
  130,              -- order
  '/bms',           -- path
  'mdi:battery-charging-100', -- icon
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  'BMS管理',
  NOW(),
  '',
  'route.bms',
  'layout.base'
);

-- 经销商管理
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  '2d8ef8ab-cf49-47eb-ae7d-a1292f6d5fc0',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_dealer',
  3,
  1301,
  '/bms/dealer',
  'mdi:account-group',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '经销商管理',
  NOW(),
  '',
  'route.bms_dealer',
  'view.bms_dealer'
);

-- 电池列表
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  'f0f4c9b7-9e9c-4f3b-a3f0-1b8c2d6c7c10',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_battery_list',
  3,
  1300,
  '/bms/battery/list',
  'mdi:battery',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '电池列表',
  NOW(),
  '',
  'route.bms_battery_list',
  'view.bms_battery_list'
);

-- BMS 看板
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  'c5a6b9d5-4cf2-4b01-9a4d-7b6240a3b7e2',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_dashboard',
  3,
  1299,
  '/bms/dashboard',
  'mdi:view-dashboard',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  'BMS 看板',
  NOW(),
  '',
  'route.bms_dashboard',
  'view.bms_dashboard'
);

-- 终端用户
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  'e2d1b1d4-8e61-4f1c-9aa7-7d8c1fb8c0d2',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_end_user',
  3,
  1300,
  '/bms/end/user',
  'mdi:account',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '终端用户',
  NOW(),
  '',
  'route.bms_end_user',
  'view.bms_end_user'
);

-- 电池型号管理
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  '55d1c47a-551b-4058-b974-3ec09d88b2d7',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_battery_model',
  3,
  1302,
  '/bms/battery/model',
  'mdi:battery-unknown',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '电池型号管理',
  NOW(),
  '',
  'route.bms_battery_model',
  'view.bms_battery_model'
);

-- 设备转移记录
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  '497266a2-48bb-4432-a7eb-dd4836e1cdaa',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_battery_transfer',
  3,
  1303,
  '/bms/battery/transfer',
  'mdi:transfer',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '设备转移记录',
  NOW(),
  '',
  'route.bms_battery_transfer',
  'view.bms_battery_transfer'
);

-- 维保中心
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  '82be3599-052d-4dfa-9e4f-d66a612ae869',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_warranty',
  3,
  1304,
  '/bms/warranty',
  'mdi:clipboard-text',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '维保中心',
  NOW(),
  '',
  'route.bms_warranty',
  'view.bms_warranty'
);

