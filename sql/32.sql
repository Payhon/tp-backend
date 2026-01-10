-- Version: 32
-- Description: Fix BMS ops menu tree (sys_ui_elements) for "运营管理" (flat: 操作记录/激活日志)

DO $$
DECLARE
    bms_root_id varchar(36);
BEGIN
    -- Ensure BMS root exists
    SELECT id INTO bms_root_id
    FROM public.sys_ui_elements
    WHERE element_code = 'bms'
    ORDER BY created_at ASC
    LIMIT 1;

    IF bms_root_id IS NULL OR bms_root_id = '' THEN
        RAISE NOTICE 'skip: bms root menu not found';
        RETURN;
    END IF;

    -- Remove broken/duplicated records (by element_code)
    DELETE FROM public.sys_ui_elements
    WHERE element_code IN (
        'bms_ops',
        'bms_ops_activation',
        'bms_ops_activation_log',
        'bms_ops_operation',
        'bms_ops_operation_log'
    );

    -- Recreate "运营管理" menu tree with stable ids (flat children)
    INSERT INTO public.sys_ui_elements (
        id, parent_id, element_code, element_type, orders,
        param1, param2, param3, authority, description,
        created_at, remark, multilingual, route_path
    ) VALUES (
        '0f7af1e2-6d4f-4b03-9f2b-8c5f9b5d9e01',
        bms_root_id,
        'bms_ops',
        2,
        1305,
        '/bms/ops/operation/log',
        'mdi:clipboard-text-outline',
        'self',
        '["TENANT_ADMIN","SYS_ADMIN"]'::json,
        '运营管理',
        NOW(),
        '',
        'route.bms_ops',
        'layout.base'
    );

    INSERT INTO public.sys_ui_elements (
        id, parent_id, element_code, element_type, orders,
        param1, param2, param3, authority, description,
        created_at, remark, multilingual, route_path
    ) VALUES (
        '2a4b7c9d-1f33-4a8e-9d31-7b5a9b2f3c22',
        '0f7af1e2-6d4f-4b03-9f2b-8c5f9b5d9e01',
        'bms_ops_operation_log',
        3,
        1306,
        '/bms/ops/operation/log',
        'mdi:clipboard-list',
        'self',
        '["TENANT_ADMIN","SYS_ADMIN"]'::json,
        '操作记录',
        NOW(),
        '',
        'route.bms_ops_operation_log',
        'view.bms_ops_operation_log'
    );

    INSERT INTO public.sys_ui_elements (
        id, parent_id, element_code, element_type, orders,
        param1, param2, param3, authority, description,
        created_at, remark, multilingual, route_path
    ) VALUES (
        '1d2f3a74-6b2a-4f4d-9b55-4d7f3d5c0a11',
        '0f7af1e2-6d4f-4b03-9f2b-8c5f9b5d9e01',
        'bms_ops_activation_log',
        3,
        1307,
        '/bms/ops/activation/log',
        'mdi:history',
        'self',
        '["TENANT_ADMIN","SYS_ADMIN"]'::json,
        '激活日志',
        NOW(),
        '',
        'route.bms_ops_activation_log',
        'view.bms_ops_activation_log'
    );
END $$;
