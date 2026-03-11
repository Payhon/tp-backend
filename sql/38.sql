-- FEAT-0009: 管理端登录二维码配置字段扩展
ALTER TABLE public.logo
    ADD COLUMN IF NOT EXISTS wxmp_qrcode varchar(255) NOT NULL DEFAULT '';

COMMENT ON COLUMN public.logo.wxmp_qrcode IS '微信小程序二维码';

ALTER TABLE public.logo
    ADD COLUMN IF NOT EXISTS app_download_qrcode varchar(255) NOT NULL DEFAULT '';

COMMENT ON COLUMN public.logo.app_download_qrcode IS 'App下载页二维码';
