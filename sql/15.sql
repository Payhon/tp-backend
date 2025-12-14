-- BMS battery maintenance records
-- Version: 15
-- Description: Add table battery_maintenance_records for manual maintenance entries

CREATE TABLE IF NOT EXISTS public.battery_maintenance_records (
  id varchar(36) NOT NULL,
  tenant_id varchar(36) NOT NULL,
  device_id varchar(36) NOT NULL,
  fault_type varchar(255) NOT NULL,
  maintain_at timestamptz NOT NULL,
  maintainer varchar(255) NOT NULL,
  solution text NULL,
  parts jsonb NULL,
  affect_warranty boolean NOT NULL DEFAULT false,
  remark varchar(255) NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT battery_maintenance_records_pk PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_battery_maintenance_records_tenant ON public.battery_maintenance_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_battery_maintenance_records_device ON public.battery_maintenance_records(device_id);
CREATE INDEX IF NOT EXISTS idx_battery_maintenance_records_maintain_at ON public.battery_maintenance_records(maintain_at);

ALTER TABLE public.battery_maintenance_records
  ADD CONSTRAINT battery_maintenance_records_devices_fk
  FOREIGN KEY (device_id) REFERENCES public.devices(id) ON DELETE CASCADE;

