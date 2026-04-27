-- FEAT-0048: BMS 4G 通讯调试管理（基于 bms-bridge）

CREATE TABLE IF NOT EXISTS public.bms_bridge_comm_logs (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    device_id VARCHAR(36) NOT NULL,
    device_number VARCHAR(64) NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'bms_bridge',
    access_mode VARCHAR(16) NOT NULL DEFAULT '4G',
    event_type VARCHAR(32) NOT NULL,
    direction VARCHAR(16) NOT NULL,
    mqtt_topic VARCHAR(255) NULL,
    qos INTEGER NULL,
    message_id VARCHAR(64) NULL,
    payload_raw TEXT NULL,
    payload_format VARCHAR(16) NULL,
    parsed_summary JSONB NULL,
    status VARCHAR(16) NOT NULL,
    error_message TEXT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bms_bridge_comm_logs_tenant_device_time
    ON public.bms_bridge_comm_logs (tenant_id, device_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_bms_bridge_comm_logs_tenant_event_time
    ON public.bms_bridge_comm_logs (tenant_id, event_type, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_bms_bridge_comm_logs_time
    ON public.bms_bridge_comm_logs (occurred_at DESC);

DO $$
DECLARE
    bms_ops_id varchar(36);
BEGIN
    SELECT id INTO bms_ops_id
    FROM public.sys_ui_elements
    WHERE element_code = 'bms_ops'
    ORDER BY created_at ASC
    LIMIT 1;

    IF bms_ops_id IS NULL OR bms_ops_id = '' THEN
        RAISE NOTICE 'skip bms_ops_comm_debug: parent bms_ops not found';
        RETURN;
    END IF;

    INSERT INTO public.sys_ui_elements (
        id, parent_id, element_code, element_type, orders,
        param1, param2, param3, authority, description,
        created_at, remark, multilingual, route_path
    )
    SELECT
        '0a13f8e2-ff04-4cc0-a0aa-0df248d0d801',
        bms_ops_id,
        'bms_ops_comm_debug',
        3,
        1308,
        '/bms/ops/comm-debug',
        'mdi:list-box-outline',
        'self',
        '["TENANT_ADMIN","SYS_ADMIN"]'::json,
        '通讯调试管理',
        NOW(),
        '',
        'route.bms_ops_comm_debug',
        'view.bms_ops_comm_debug'
    WHERE NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_ops_comm_debug'
    );

    UPDATE public.sys_ui_elements
    SET
        parent_id = bms_ops_id,
        element_type = 3,
        orders = 1308,
        param1 = '/bms/ops/comm-debug',
        param2 = 'mdi:list-box-outline',
        param3 = 'self',
        authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
        description = '通讯调试管理',
        multilingual = 'route.bms_ops_comm_debug',
        route_path = 'view.bms_ops_comm_debug'
    WHERE element_code = 'bms_ops_comm_debug';
END $$;
