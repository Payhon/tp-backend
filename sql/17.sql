-- BMS battery tags
-- Version: 17
-- Description: Add battery tag management and device-tag relations

CREATE TABLE IF NOT EXISTS public.battery_tags (
  id varchar(36) NOT NULL,
  tenant_id varchar(36) NOT NULL,
  name varchar(64) NOT NULL,
  color varchar(32) NULL,
  scene varchar(64) NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT battery_tags_pk PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_battery_tags_tenant_name ON public.battery_tags(tenant_id, name);
CREATE INDEX IF NOT EXISTS idx_battery_tags_tenant ON public.battery_tags(tenant_id);

CREATE TABLE IF NOT EXISTS public.device_battery_tags (
  id varchar(36) NOT NULL,
  tenant_id varchar(36) NOT NULL,
  device_id varchar(36) NOT NULL,
  tag_id varchar(36) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT device_battery_tags_pk PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_device_battery_tags_device_tag ON public.device_battery_tags(device_id, tag_id);
CREATE INDEX IF NOT EXISTS idx_device_battery_tags_tenant ON public.device_battery_tags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_device_battery_tags_device ON public.device_battery_tags(device_id);
CREATE INDEX IF NOT EXISTS idx_device_battery_tags_tag ON public.device_battery_tags(tag_id);

ALTER TABLE public.device_battery_tags
  ADD CONSTRAINT device_battery_tags_devices_fk
  FOREIGN KEY (device_id) REFERENCES public.devices(id) ON DELETE CASCADE;

ALTER TABLE public.device_battery_tags
  ADD CONSTRAINT device_battery_tags_battery_tags_fk
  FOREIGN KEY (tag_id) REFERENCES public.battery_tags(id) ON DELETE CASCADE;

-- Menu: 标签管理（挂在 BMS 菜单下；前端将其归入“电池管理”分组）
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  'd3b0b08d-7d74-4c5d-ae7f-1a2b3c4d5eaa',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_battery_tag',
  3,
  1308,
  '/bms/battery/tag',
  'mdi:tag',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '标签管理',
  NOW(),
  '',
  'route.bms_battery_tag',
  'view.bms_battery_tag'
);

