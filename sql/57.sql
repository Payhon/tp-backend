-- FEAT-0055: PACK 厂微信小程序配置接入

CREATE TABLE IF NOT EXISTS public.pack_wxmp_configs (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	org_id varchar(36) NOT NULL,
	app_id varchar(36) NOT NULL,
	wx_appid varchar(100) NOT NULL,
	app_secret varchar(128) NOT NULL,
	status varchar(16) NOT NULL DEFAULT 'OPEN',
	home_banner_url varchar(500) NULL,
	login_logo_url varchar(500) NULL,
	remark varchar(255) NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT pack_wxmp_configs_pkey PRIMARY KEY (id),
	CONSTRAINT pack_wxmp_configs_org_fk FOREIGN KEY (org_id) REFERENCES public.orgs(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT pack_wxmp_configs_app_fk FOREIGN KEY (app_id) REFERENCES public.apps(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE public.pack_wxmp_configs IS 'PACK 厂微信小程序配置';
COMMENT ON COLUMN public.pack_wxmp_configs.org_id IS 'PACK 厂组织ID';
COMMENT ON COLUMN public.pack_wxmp_configs.app_id IS '关联 apps.id，用于内容页配置';
COMMENT ON COLUMN public.pack_wxmp_configs.wx_appid IS '微信小程序 AppID';
COMMENT ON COLUMN public.pack_wxmp_configs.app_secret IS '微信小程序 AppSecret';
COMMENT ON COLUMN public.pack_wxmp_configs.status IS '状态: OPEN/CLOSE';
COMMENT ON COLUMN public.pack_wxmp_configs.home_banner_url IS '小程序首页 Banner 图片 URL';
COMMENT ON COLUMN public.pack_wxmp_configs.login_logo_url IS '小程序登录页 Logo 图片 URL';

CREATE UNIQUE INDEX IF NOT EXISTS uk_pack_wxmp_configs_tenant_org
	ON public.pack_wxmp_configs (tenant_id, org_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_pack_wxmp_configs_tenant_wx_appid
	ON public.pack_wxmp_configs (tenant_id, wx_appid);
CREATE INDEX IF NOT EXISTS idx_pack_wxmp_configs_tenant_app
	ON public.pack_wxmp_configs (tenant_id, app_id);
