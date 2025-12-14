-- BMS offline command tasks
-- Version: 18
-- Description: Store offline commands and execute when device comes online

CREATE TABLE IF NOT EXISTS public.offline_command_tasks (
  id varchar(36) NOT NULL,
  tenant_id varchar(36) NOT NULL,
  device_id varchar(36) NOT NULL,
  device_number varchar(64) NOT NULL,
  command_type varchar(64) NOT NULL, -- 展示用：重启/休眠/清除告警/OTA升级 或 物模型命令名称
  identify varchar(255) NOT NULL,    -- 下发命令标识符（对应 PutMessageForCommand.identify）
  payload text NULL,                -- 对应 PutMessageForCommand.value（JSON 字符串）
  created_by varchar(36) NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  status varchar(16) NOT NULL DEFAULT 'PENDING', -- PENDING/SENT/SUCCESS/FAILED/CANCELLED
  dispatched_at timestamptz NULL,
  executed_at timestamptz NULL,
  message_id varchar(32) NULL,
  error_message text NULL,
  CONSTRAINT offline_command_tasks_pk PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_offline_command_tasks_tenant ON public.offline_command_tasks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_offline_command_tasks_device ON public.offline_command_tasks(device_id);
CREATE INDEX IF NOT EXISTS idx_offline_command_tasks_status ON public.offline_command_tasks(status);
CREATE INDEX IF NOT EXISTS idx_offline_command_tasks_created_at ON public.offline_command_tasks(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS ux_offline_command_tasks_message_id ON public.offline_command_tasks(message_id) WHERE message_id IS NOT NULL;

ALTER TABLE public.offline_command_tasks
  ADD CONSTRAINT offline_command_tasks_devices_fk
  FOREIGN KEY (device_id) REFERENCES public.devices(id) ON DELETE CASCADE;

-- Menu: 离线指令（挂在 BMS -> 电池管理 下）
INSERT INTO public.sys_ui_elements (
  id, parent_id, element_code, element_type, orders,
  param1, param2, param3, authority, description,
  created_at, remark, multilingual, route_path
)
VALUES (
  '7b1d1df1-2d26-4b24-9c51-0b6a3c8d4f11',
  'a753c525-780f-415f-a2b6-3d909c79f7f6',
  'bms_battery_offline_cmd',
  3,
  1310,
  '/bms/battery/offline-command',
  'mdi:cloud-clock',
  'self',
  '["TENANT_ADMIN","SYS_ADMIN"]'::json,
  '离线指令',
  NOW(),
  '',
  'route.bms_battery_offline_cmd',
  'view.bms_battery_offline_cmd'
);

