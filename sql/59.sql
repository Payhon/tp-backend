-- FEAT-0056: device_batteries add protocol identity BLE MAC for external BLE passthrough modules

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'device_batteries'
          AND column_name = 'identity_ble_mac'
    ) THEN
        ALTER TABLE public.device_batteries ADD COLUMN identity_ble_mac VARCHAR(32);
        COMMENT ON COLUMN public.device_batteries.identity_ble_mac IS 'BMS协议身份区蓝牙MAC地址（区别于连接/广播用ble_mac）';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_device_batteries_identity_ble_mac
    ON public.device_batteries (identity_ble_mac);
