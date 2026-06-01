-- FEAT-0055: PACK 厂微信小程序登录 Logo 配置
ALTER TABLE public.pack_wxmp_configs
	ADD COLUMN IF NOT EXISTS login_logo_url varchar(500) NULL;

COMMENT ON COLUMN public.pack_wxmp_configs.login_logo_url IS '小程序登录页 Logo 图片 URL';
