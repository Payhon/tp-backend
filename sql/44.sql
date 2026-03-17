-- FEAT-0014: 电池信息补全（电芯品牌/电池型号）

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'device_batteries' AND column_name = 'cell_brand_seq_no'
    ) THEN
        ALTER TABLE public.device_batteries ADD COLUMN cell_brand_seq_no smallint;
        COMMENT ON COLUMN public.device_batteries.cell_brand_seq_no IS '电芯品牌序号（关联 battery_cell_brands.seq_no）';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'device_batteries' AND column_name = 'battery_model_seq_no'
    ) THEN
        ALTER TABLE public.device_batteries ADD COLUMN battery_model_seq_no smallint;
        COMMENT ON COLUMN public.device_batteries.battery_model_seq_no IS '电池型号序号（关联 battery_models.seq_no）';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_device_batteries_cell_brand_seq_no
    ON public.device_batteries (cell_brand_seq_no);

CREATE INDEX IF NOT EXISTS idx_device_batteries_battery_model_seq_no
    ON public.device_batteries (battery_model_seq_no);

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

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements
        WHERE element_code = 'bms_battery_list_batch_info_complete'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1015', battery_list_id, 'bms_battery_list_batch_info_complete', 4, 13023,
            'bms_battery_list_batch_info_complete', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '批量电池信息补全', NOW(), '页面元素权限', 'perm.bms_battery_list_batch_info_complete', ''
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements
        WHERE element_code = 'bms_battery_list_batch_assign_dealer'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1017', battery_list_id, 'bms_battery_list_batch_assign_dealer', 4, 13024,
            'bms_battery_list_batch_assign_dealer', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '批量分配经销商', NOW(), '页面元素权限', 'perm.bms_battery_list_batch_assign_dealer', ''
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements
        WHERE element_code = 'bms_battery_list_batch_tag'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1018', battery_list_id, 'bms_battery_list_batch_tag', 4, 13025,
            'bms_battery_list_batch_tag', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '批量设置标签', NOW(), '页面元素权限', 'perm.bms_battery_list_batch_tag', ''
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements
        WHERE element_code = 'bms_battery_list_batch_command'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1019', battery_list_id, 'bms_battery_list_batch_command', 4, 13026,
            'bms_battery_list_batch_command', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '批量下发指令', NOW(), '页面元素权限', 'perm.bms_battery_list_batch_command', ''
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements
        WHERE element_code = 'bms_battery_list_batch_ota'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1020', battery_list_id, 'bms_battery_list_batch_ota', 4, 13027,
            'bms_battery_list_batch_ota', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '批量OTA推送', NOW(), '页面元素权限', 'perm.bms_battery_list_batch_ota', ''
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements
        WHERE element_code = 'bms_battery_list_action_lifecycle_info_complete'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1016', battery_list_id, 'bms_battery_list_action_lifecycle_info_complete', 4, 13034,
            'bms_battery_list_action_lifecycle_info_complete', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '生命周期-信息补全', NOW(), '页面元素权限', 'perm.bms_battery_list_action_lifecycle_info_complete', ''
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements
        WHERE element_code = 'bms_battery_list_action_edit_bms_info'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1021', battery_list_id, 'bms_battery_list_action_edit_bms_info', 4, 13035,
            'bms_battery_list_action_edit_bms_info', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '编辑BMS信息', NOW(), '页面元素权限', 'perm.bms_battery_list_action_edit_bms_info', ''
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements
        WHERE element_code = 'bms_battery_list_action_delete'
    ) THEN
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
        ) VALUES (
            'b9d0a501-6d9d-4de0-a2eb-8f4fd15f1022', battery_list_id, 'bms_battery_list_action_delete', 4, 13036,
            'bms_battery_list_action_delete', '', '1', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '删除', NOW(), '页面元素权限', 'perm.bms_battery_list_action_delete', ''
        );
    END IF;
END $$;

WITH target_rows AS (
    SELECT tenant_id, org_type, COALESCE(ui_codes, '[]'::jsonb) AS ui_codes
    FROM public.org_type_permissions
    WHERE org_type = 'PACK_FACTORY'
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
                    UNION ALL SELECT 'bms_battery_list_batch_info_complete'
                    UNION ALL SELECT 'bms_battery_list_batch_assign_dealer'
                    UNION ALL SELECT 'bms_battery_list_batch_tag'
                    UNION ALL SELECT 'bms_battery_list_batch_command'
                    UNION ALL SELECT 'bms_battery_list_batch_ota'
                    UNION ALL SELECT 'bms_battery_list_action_lifecycle_info_complete'
                    UNION ALL SELECT 'bms_battery_list_action_edit_bms_info'
                    UNION ALL SELECT 'bms_battery_list_action_delete'
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
