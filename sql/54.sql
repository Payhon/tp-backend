-- FEAT-0051: 4G module OTA package management

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'ota_upgrade_packages'
          AND column_name = 'device_kind'
    ) THEN
        ALTER TABLE public.ota_upgrade_packages
            ADD COLUMN device_kind int2 NOT NULL DEFAULT 1;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'ota_upgrade_packages'
          AND column_name = 'is_latest'
    ) THEN
        ALTER TABLE public.ota_upgrade_packages
            ADD COLUMN is_latest boolean NOT NULL DEFAULT false;
    END IF;

    COMMENT ON COLUMN public.ota_upgrade_packages.device_kind IS '设备类型 1-BMS 2-仪表 3-4G模块';
    COMMENT ON COLUMN public.ota_upgrade_packages.is_latest IS '是否最新固件';
END $$;

CREATE INDEX IF NOT EXISTS idx_ota_upgrade_packages_tenant_device_kind
    ON public.ota_upgrade_packages (tenant_id, device_kind);

CREATE UNIQUE INDEX IF NOT EXISTS uq_ota_upgrade_packages_4g_latest_per_tenant
    ON public.ota_upgrade_packages (tenant_id)
    WHERE device_kind = 3 AND is_latest IS TRUE AND tenant_id IS NOT NULL;
