-- FEAT-0013: 当前用户在本次设备绑定内的自定义名称，解绑时随绑定行删除。
ALTER TABLE public.device_user_bindings
ADD COLUMN IF NOT EXISTS custom_name varchar(255) NULL;

COMMENT ON COLUMN public.device_user_bindings.custom_name IS '当前绑定用户自定义设备名称';
