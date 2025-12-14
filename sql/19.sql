-- BMS OTA menus
-- Version: 19
-- Description: Register BMS OTA package/task pages

-- OTA升级包管理
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  '2d6b8f0f-7b6d-4f5c-9d85-1f2e3a4b5c6d',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_battery_ota_package',
  3,
  1312,
  '/bms/battery/ota/package',
  'mdi:package-variant',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  'OTA升级包管理',
  NOW(),
  '',
  'route.bms_battery_ota_package',
  'view.bms_battery_ota_package'
);

-- OTA升级任务管理
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  '7a2c3d4e-5f60-4a1b-9c2d-1e0f9a8b7c6d',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_battery_ota_task',
  3,
  1313,
  '/bms/battery/ota/task',
  'mdi:clipboard-list-outline',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  'OTA升级任务管理',
  NOW(),
  '',
  'route.bms_battery_ota_task',
  'view.bms_battery_ota_task'
);

