-- FEAT-0053: BMS OTA package constraints

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'ota_upgrade_packages'
          AND column_name = 'battery_model_id'
    ) THEN
        ALTER TABLE public.ota_upgrade_packages
            ADD COLUMN battery_model_id varchar(36);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'ota_upgrade_packages'
          AND column_name = 'batch_number'
    ) THEN
        ALTER TABLE public.ota_upgrade_packages
            ADD COLUMN batch_number varchar(100);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'ota_upgrade_packages'
          AND column_name = 'item_uuid'
    ) THEN
        ALTER TABLE public.ota_upgrade_packages
            ADD COLUMN item_uuid varchar(64);
    END IF;

    COMMENT ON COLUMN public.ota_upgrade_packages.battery_model_id IS 'BMS型号ID约束';
    COMMENT ON COLUMN public.ota_upgrade_packages.batch_number IS '批号约束';
    COMMENT ON COLUMN public.ota_upgrade_packages.item_uuid IS '序列号约束，对应 device_batteries.item_uuid';
END $$;

CREATE INDEX IF NOT EXISTS idx_ota_upgrade_packages_tenant_kind_item_uuid
    ON public.ota_upgrade_packages (tenant_id, device_kind, item_uuid);

CREATE INDEX IF NOT EXISTS idx_ota_upgrade_packages_tenant_kind_battery_model
    ON public.ota_upgrade_packages (tenant_id, device_kind, battery_model_id);

CREATE INDEX IF NOT EXISTS idx_ota_upgrade_packages_tenant_kind_batch_number
    ON public.ota_upgrade_packages (tenant_id, device_kind, batch_number);
