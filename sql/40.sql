-- FEAT-0011: 电池列表详情路由与页面元素权限补齐（机构用户）

DO $$
DECLARE
  bms_root_id varchar(36);
  battery_list_id varchar(36);
BEGIN
  SELECT id INTO bms_root_id
  FROM public.sys_ui_elements
  WHERE element_code = 'bms'
  LIMIT 1;

  IF bms_root_id IS NULL THEN
    bms_root_id := 'a753c525-780f-415f-a2b6-3d909c79f7f6';
  END IF;

  SELECT id INTO battery_list_id
  FROM public.sys_ui_elements
  WHERE element_code = 'bms_battery_list'
  LIMIT 1;

  IF battery_list_id IS NULL THEN
    battery_list_id := 'f0f4c9b7-9e9c-4f3b-a3f0-1b8c2d6c7c10';
  END IF;

  -- 历史数据兼容：电池列表必须是页面路由，避免被当成目录导致页面空白
  UPDATE public.sys_ui_elements
  SET
    element_type = 3,
    param1 = '/bms/battery/list',
    route_path = 'view.bms_battery_list'
  WHERE element_code = 'bms_battery_list';

  -- 电池列表 -> 详情页路由（隐藏菜单，仅作为可访问路由权限）
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1001',
    bms_root_id,
    'bms_battery_list_detail',
    3,
    13001,
    '/device/details',
    'mdi:file-document-outline',
    '1',
    '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '电池详情页',
    NOW(),
    '电池列表操作->查看详情路由权限',
    'route.device_details',
    'view.device_details'
  WHERE NOT EXISTS (
    SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_detail'
  );

  UPDATE public.sys_ui_elements
  SET
    parent_id = bms_root_id,
    element_type = 3,
    orders = 13001,
    param1 = '/device/details',
    param2 = 'mdi:file-document-outline',
    param3 = '1',
    authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    description = '电池详情页',
    multilingual = 'route.device_details',
    route_path = 'view.device_details'
  WHERE element_code = 'bms_battery_list_detail';

  -- 顶部按钮：导出
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1002', battery_list_id,
    'bms_battery_list_export', 4, 13011,
    'bms_battery_list_export', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '导出', NOW(), '页面元素权限', 'perm.bms_battery_list_export', ''
  WHERE NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_export');

  -- 顶部按钮：新增BMS
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1003', battery_list_id,
    'bms_battery_list_add', 4, 13012,
    'bms_battery_list_add', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '新增BMS', NOW(), '页面元素权限', 'perm.bms_battery_list_add', ''
  WHERE NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_add');

  -- 顶部按钮：导入
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1004', battery_list_id,
    'bms_battery_list_import', 4, 13013,
    'bms_battery_list_import', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '导入', NOW(), '页面元素权限', 'perm.bms_battery_list_import', ''
  WHERE NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_import');

  -- 操作菜单：参数
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1005', battery_list_id,
    'bms_battery_list_action_params', 4, 13021,
    'bms_battery_list_action_params', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '参数', NOW(), '页面元素权限', 'perm.bms_battery_list_action_params', ''
  WHERE NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_action_params');

  -- 操作菜单：离线指令
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1006', battery_list_id,
    'bms_battery_list_action_offline_command', 4, 13022,
    'bms_battery_list_action_offline_command', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '离线指令', NOW(), '页面元素权限', 'perm.bms_battery_list_action_offline_command', ''
  WHERE NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_action_offline_command');

  -- 生命周期：出厂
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1007', battery_list_id,
    'bms_battery_list_action_lifecycle_factory', 4, 13031,
    'bms_battery_list_action_lifecycle_factory', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '生命周期-出厂', NOW(), '页面元素权限', 'perm.bms_battery_list_action_lifecycle_factory', ''
  WHERE NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_action_lifecycle_factory');

  -- 生命周期：激活
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1008', battery_list_id,
    'bms_battery_list_action_lifecycle_activate', 4, 13032,
    'bms_battery_list_action_lifecycle_activate', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '生命周期-激活', NOW(), '页面元素权限', 'perm.bms_battery_list_action_lifecycle_activate', ''
  WHERE NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_action_lifecycle_activate');

  -- 生命周期：调拨
  INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
  )
  SELECT
    'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1009', battery_list_id,
    'bms_battery_list_action_lifecycle_transfer', 4, 13033,
    'bms_battery_list_action_lifecycle_transfer', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '生命周期-调拨', NOW(), '页面元素权限', 'perm.bms_battery_list_action_lifecycle_transfer', ''
  WHERE NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_battery_list_action_lifecycle_transfer');

  -- 修正已存在按钮权限的显示名称（避免“菜单管理”表格显示 i18n key）
  UPDATE public.sys_ui_elements
  SET description = '导出', multilingual = 'perm.bms_battery_list_export'
  WHERE element_code = 'bms_battery_list_export';

  UPDATE public.sys_ui_elements
  SET description = '新增BMS', multilingual = 'perm.bms_battery_list_add'
  WHERE element_code = 'bms_battery_list_add';

  UPDATE public.sys_ui_elements
  SET description = '导入', multilingual = 'perm.bms_battery_list_import'
  WHERE element_code = 'bms_battery_list_import';

  UPDATE public.sys_ui_elements
  SET description = '参数', multilingual = 'perm.bms_battery_list_action_params'
  WHERE element_code = 'bms_battery_list_action_params';

  UPDATE public.sys_ui_elements
  SET description = '离线指令', multilingual = 'perm.bms_battery_list_action_offline_command'
  WHERE element_code = 'bms_battery_list_action_offline_command';

  UPDATE public.sys_ui_elements
  SET description = '生命周期-出厂', multilingual = 'perm.bms_battery_list_action_lifecycle_factory'
  WHERE element_code = 'bms_battery_list_action_lifecycle_factory';

  UPDATE public.sys_ui_elements
  SET description = '生命周期-激活', multilingual = 'perm.bms_battery_list_action_lifecycle_activate'
  WHERE element_code = 'bms_battery_list_action_lifecycle_activate';

  UPDATE public.sys_ui_elements
  SET description = '生命周期-调拨', multilingual = 'perm.bms_battery_list_action_lifecycle_transfer'
  WHERE element_code = 'bms_battery_list_action_lifecycle_transfer';
END $$;

-- 为已授予“电池列表”的机构类型自动补齐详情/按钮权限编码
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
          UNION ALL SELECT 'bms_battery_list_detail'
          UNION ALL SELECT 'bms_battery_list_export'
          UNION ALL SELECT 'bms_battery_list_add'
          UNION ALL SELECT 'bms_battery_list_import'
          UNION ALL SELECT 'bms_battery_list_action_params'
          UNION ALL SELECT 'bms_battery_list_action_offline_command'
          UNION ALL SELECT 'bms_battery_list_action_lifecycle_factory'
          UNION ALL SELECT 'bms_battery_list_action_lifecycle_activate'
          UNION ALL SELECT 'bms_battery_list_action_lifecycle_transfer'
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
