-- Version: 29
-- Description: device_batteries add item_uuid/ble_mac/comm_chip_id for mobile app

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='item_uuid') THEN
        ALTER TABLE device_batteries ADD COLUMN item_uuid VARCHAR(64);
        COMMENT ON COLUMN device_batteries.item_uuid IS '电池主板序列号ID（作为设备唯一ID）';
        CREATE INDEX IF NOT EXISTS idx_device_batteries_item_uuid ON device_batteries(item_uuid);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='ble_mac') THEN
        ALTER TABLE device_batteries ADD COLUMN ble_mac VARCHAR(32);
        COMMENT ON COLUMN device_batteries.ble_mac IS '设备蓝牙芯片MAC地址（APP优先BLE连接）';
        CREATE INDEX IF NOT EXISTS idx_device_batteries_ble_mac ON device_batteries(ble_mac);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='comm_chip_id') THEN
        ALTER TABLE device_batteries ADD COLUMN comm_chip_id VARCHAR(64);
        COMMENT ON COLUMN device_batteries.comm_chip_id IS '4G通讯芯片ID';
        CREATE INDEX IF NOT EXISTS idx_device_batteries_comm_chip_id ON device_batteries(comm_chip_id);
    END IF;
END $$;

