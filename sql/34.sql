-- Description: device_batteries add 4G socket fields (0x900~0x923)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='longitude') THEN
        ALTER TABLE device_batteries ADD COLUMN longitude DOUBLE PRECISION;
        COMMENT ON COLUMN device_batteries.longitude IS '经度(WGS84, 0x900, 分辨率0.00001)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='latitude') THEN
        ALTER TABLE device_batteries ADD COLUMN latitude DOUBLE PRECISION;
        COMMENT ON COLUMN device_batteries.latitude IS '纬度(WGS84, 0x902, 分辨率0.00001)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='speed') THEN
        ALTER TABLE device_batteries ADD COLUMN speed DOUBLE PRECISION;
        COMMENT ON COLUMN device_batteries.speed IS '速度(km/h, 0x904, 分辨率0.001)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='altitude') THEN
        ALTER TABLE device_batteries ADD COLUMN altitude INTEGER;
        COMMENT ON COLUMN device_batteries.altitude IS '高度(m, 0x905)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='rssi') THEN
        ALTER TABLE device_batteries ADD COLUMN rssi INTEGER;
        COMMENT ON COLUMN device_batteries.rssi IS '信号强度RSSI(0x906)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='tac') THEN
        ALTER TABLE device_batteries ADD COLUMN tac INTEGER;
        COMMENT ON COLUMN device_batteries.tac IS '位置区码TAC(0x907)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='cell_id') THEN
        ALTER TABLE device_batteries ADD COLUMN cell_id BIGINT;
        COMMENT ON COLUMN device_batteries.cell_id IS '小区识别码Cell ID(0x908)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='imei') THEN
        ALTER TABLE device_batteries ADD COLUMN imei VARCHAR(18);
        COMMENT ON COLUMN device_batteries.imei IS 'IMEI(0x90A~0x912)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='iccid') THEN
        ALTER TABLE device_batteries ADD COLUMN iccid VARCHAR(22);
        COMMENT ON COLUMN device_batteries.iccid IS 'ICCID(0x913~0x91D)';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='module_sw_version') THEN
        ALTER TABLE device_batteries ADD COLUMN module_sw_version VARCHAR(32);
        COMMENT ON COLUMN device_batteries.module_sw_version IS '4G模块软件版本号(0x91E~0x923)';
    END IF;
END $$;
