-- Version: 34
-- Description: device_batteries add bms_comm_type (1:BLE,2:4G,3:BLE+4G)

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='bms_comm_type') THEN
        ALTER TABLE device_batteries ADD COLUMN bms_comm_type INT;
        COMMENT ON COLUMN device_batteries.bms_comm_type IS 'BMS通讯类型：1蓝牙、2:4G、3:蓝牙+4G';
        CREATE INDEX IF NOT EXISTS idx_device_batteries_bms_comm_type ON device_batteries(bms_comm_type);
    END IF;
END $$;

