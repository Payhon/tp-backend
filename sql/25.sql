-- Version: 25
-- Description: APP管理模块（apps/app_versions）+ 菜单（sys_ui_elements）

-- ============================================================================
-- 1. apps 表
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.apps (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	appid varchar(100) NOT NULL,
	app_type int2 NOT NULL DEFAULT 0, -- 0: uni-app, 1: uni-app x
	name varchar(255) NOT NULL,
	description text NULL,
	creator_uid varchar(36) NULL,
	owner_type int2 NULL, -- 1: 个人, 2: 企业
	owner_id varchar(36) NULL,
	managers jsonb NOT NULL DEFAULT '[]'::jsonb,
	members jsonb NOT NULL DEFAULT '[]'::jsonb,
	icon_url varchar(500) NULL,
	introduction text NULL,
	screenshot jsonb NOT NULL DEFAULT '[]'::jsonb,
	app_android jsonb NULL,
	app_ios jsonb NULL,
	app_harmony jsonb NULL,
	mp_weixin jsonb NULL,
	mp_alipay jsonb NULL,
	mp_baidu jsonb NULL,
	mp_toutiao jsonb NULL,
	mp_qq jsonb NULL,
	mp_kuaishou jsonb NULL,
	mp_lark jsonb NULL,
	mp_jd jsonb NULL,
	mp_dingtalk jsonb NULL,
	h5 jsonb NULL,
	quickapp jsonb NULL,
	store_list jsonb NOT NULL DEFAULT '[]'::jsonb,
	remark text NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT apps_pkey PRIMARY KEY (id)
);

COMMENT ON TABLE public.apps IS 'APP信息（升级中心/应用管理）';
COMMENT ON COLUMN public.apps.appid IS '应用 AppID';
COMMENT ON COLUMN public.apps.app_type IS '应用类型：0-uni-app 1-uni-app x';
COMMENT ON COLUMN public.apps.creator_uid IS '创建者用户ID（创建者不随转让变化）';
COMMENT ON COLUMN public.apps.owner_type IS '归属者类型：1-个人 2-企业';
COMMENT ON COLUMN public.apps.owner_id IS '归属者ID（user_id or enterprise_id）';

CREATE UNIQUE INDEX IF NOT EXISTS uk_apps_tenant_appid ON public.apps (tenant_id, appid);
CREATE INDEX IF NOT EXISTS idx_apps_tenant_id ON public.apps (tenant_id);
CREATE INDEX IF NOT EXISTS idx_apps_created_at ON public.apps (created_at DESC);

-- ============================================================================
-- 2. app_versions 表
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.app_versions (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	app_id varchar(36) NOT NULL,
	appid varchar(100) NOT NULL,
	name varchar(255) NOT NULL,
	title varchar(255) NULL,
	contents text NULL,
	platform jsonb NOT NULL DEFAULT '[]'::jsonb, -- ["Android","iOS","Harmony"]
	type varchar(20) NOT NULL, -- native_app / wgt
	version varchar(50) NOT NULL,
	min_uni_version varchar(50) NULL,
	url varchar(500) NULL,
	stable_publish bool NOT NULL DEFAULT false,
	is_silently bool NOT NULL DEFAULT false,
	is_mandatory bool NOT NULL DEFAULT false,
	create_date timestamptz NOT NULL DEFAULT NOW(),
	uni_platform varchar(50) NOT NULL,
	create_env varchar(50) NOT NULL,
	store_list jsonb NOT NULL DEFAULT '[]'::jsonb,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT app_versions_pkey PRIMARY KEY (id),
	CONSTRAINT app_versions_apps_fk FOREIGN KEY (app_id) REFERENCES public.apps(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE public.app_versions IS 'APP版本信息（升级中心）';
COMMENT ON COLUMN public.app_versions.type IS '安装包类型：native_app/wgt';
COMMENT ON COLUMN public.app_versions.stable_publish IS '是否上线发行';
COMMENT ON COLUMN public.app_versions.create_env IS '创建来源：uni-stat/upgrade-center';

CREATE INDEX IF NOT EXISTS idx_app_versions_tenant_app ON public.app_versions (tenant_id, app_id);
CREATE INDEX IF NOT EXISTS idx_app_versions_create_date ON public.app_versions (create_date DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uk_app_versions_app_version_type_uni_platform
	ON public.app_versions (tenant_id, app_id, version, type, uni_platform);

-- ============================================================================
-- 3. 菜单：APP管理（sys_ui_elements）
-- ============================================================================
DO $$
DECLARE
	root_id varchar(36) := 'b9c8f7b5-6ef0-4c1b-9b6f-6c2c3a5a5c25';
BEGIN
	-- Root
	IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'app') THEN
		INSERT INTO public.sys_ui_elements (
			id, parent_id, element_code, element_type, orders,
			param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
		) VALUES (
			root_id, '0', 'app', 1, 80,
			'/app', 'mdi:cellphone-cog', '0', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
			'APP管理', NOW(), '', 'route.app_manage', 'layout.base'
		);
	END IF;

	-- 应用管理
	IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'app_manage_apps') THEN
		INSERT INTO public.sys_ui_elements (
			id, parent_id, element_code, element_type, orders,
			param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
		) VALUES (
			'0d4f9f1f-57df-4c58-9f51-3dbbb1f7c2d3', root_id, 'app_manage_apps', 3, 801,
			'/app/manage/apps', 'mdi:apps', '0', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
			'应用管理', NOW(), '', 'route.app_manage_apps', 'view.app_manage_apps'
		);
	END IF;

	-- App升级中心
	IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'app_manage_upgrade') THEN
		INSERT INTO public.sys_ui_elements (
			id, parent_id, element_code, element_type, orders,
			param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
		) VALUES (
			'2b8c3b0d-6f2e-4d4e-93d0-7a4a2b7b54c1', root_id, 'app_manage_upgrade', 3, 802,
			'/app/manage/upgrade', 'mdi:cloud-upload', '0', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
			'App升级中心', NOW(), '', 'route.app_manage_upgrade', 'view.app_manage_upgrade'
		);
	END IF;

	-- 用户管理
	IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'app_manage_users') THEN
		INSERT INTO public.sys_ui_elements (
			id, parent_id, element_code, element_type, orders,
			param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
		) VALUES (
			'64c3a6a4-5c92-4f3a-90ab-4c5f7b6f9c18', root_id, 'app_manage_users', 3, 803,
			'/app/manage/users', 'mdi:account-group', '0', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
			'用户管理', NOW(), '', 'route.app_manage_users', 'view.app_manage_users'
		);
	END IF;

	-- Ensure paths are consistent (idempotent updates)
	UPDATE public.sys_ui_elements SET param1 = '/app' WHERE element_code = 'app';
	UPDATE public.sys_ui_elements SET param1 = '/app/manage/apps' WHERE element_code = 'app_manage_apps';
	UPDATE public.sys_ui_elements SET param1 = '/app/manage/upgrade' WHERE element_code = 'app_manage_upgrade';
	UPDATE public.sys_ui_elements SET param1 = '/app/manage/users' WHERE element_code = 'app_manage_users';
END $$;
