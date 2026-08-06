ALTER TABLE public.devices
ADD COLUMN IF NOT EXISTS last_connected_at timestamptz(6) NULL;

COMMENT ON COLUMN public.devices.last_connected_at IS '最近一次成功通讯时间（服务端时间）';
