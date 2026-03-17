-- Version: 43
-- Description: add app_device_added_records for org mobile device visibility

CREATE TABLE IF NOT EXISTS app_device_added_records (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    device_id VARCHAR(36) NOT NULL,
    source VARCHAR(32) DEFAULT 'BLE_SCAN',
    added_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    remark VARCHAR(255)
);

COMMENT ON TABLE app_device_added_records IS '机构账号移动端“我添加的设备”记录表';
COMMENT ON COLUMN app_device_added_records.tenant_id IS '租户ID';
COMMENT ON COLUMN app_device_added_records.user_id IS '机构账号ID';
COMMENT ON COLUMN app_device_added_records.device_id IS '设备ID';
COMMENT ON COLUMN app_device_added_records.source IS '添加来源(BLE_SCAN/UUID_SCAN)';
COMMENT ON COLUMN app_device_added_records.added_at IS '首次添加时间';
COMMENT ON COLUMN app_device_added_records.last_seen_at IS '最近一次添加/确认时间';

CREATE UNIQUE INDEX IF NOT EXISTS idx_app_device_added_records_user_device
    ON app_device_added_records(tenant_id, user_id, device_id);

CREATE INDEX IF NOT EXISTS idx_app_device_added_records_user_added_at
    ON app_device_added_records(tenant_id, user_id, added_at DESC);

CREATE INDEX IF NOT EXISTS idx_app_device_added_records_device_id
    ON app_device_added_records(device_id);
