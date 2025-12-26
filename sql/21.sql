-- ✅2025/12/25 APP账号体系：多身份登录、模板配置、微信小程序配置

-- ============================================================================
-- 1) user_identities：一用户多账号身份（phone/email/wxmp_openid）
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.user_identities (
	id varchar(36) NOT NULL,
	user_id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	identity_type varchar(32) NOT NULL,      -- PHONE / EMAIL / WXMP_OPENID
	identifier varchar(255) NOT NULL,        -- 账号标识（手机号/邮箱/openid）
	credential_type varchar(32) NOT NULL,    -- PASSWORD / CODE
	password_hash varchar(255) NULL,         -- 密码hash（credential_type=PASSWORD）
	verified_at timestamptz NULL,            -- 验证通过时间（绑定/注册/换绑）
	is_primary bool NOT NULL DEFAULT false,  -- 是否主账号（用于展示/找回）
	status varchar(16) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE / DISABLED
	extra json NULL DEFAULT '{}'::json,      -- 扩展信息（例如微信session_key等不建议长期保存）
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT user_identities_pkey PRIMARY KEY (id)
);

COMMENT ON TABLE public.user_identities IS '用户多身份账号表（APP/小程序登录体系）';
COMMENT ON COLUMN public.user_identities.identity_type IS '身份类型: PHONE/EMAIL/WXMP_OPENID';
COMMENT ON COLUMN public.user_identities.identifier IS '身份标识：手机号/邮箱/openid';
COMMENT ON COLUMN public.user_identities.credential_type IS '凭据类型: PASSWORD/CODE';
COMMENT ON COLUMN public.user_identities.password_hash IS '密码hash';
COMMENT ON COLUMN public.user_identities.is_primary IS '是否主身份';
COMMENT ON COLUMN public.user_identities.status IS '状态: ACTIVE/DISABLED';

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_identities_tenant_type_identifier
	ON public.user_identities (tenant_id, identity_type, identifier);
CREATE INDEX IF NOT EXISTS idx_user_identities_user_id
	ON public.user_identities (user_id);

-- ============================================================================
-- 2) wx_mp_apps：微信小程序配置（按租户）
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.wx_mp_apps (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	appid varchar(64) NOT NULL,
	app_secret varchar(128) NOT NULL,
	status varchar(16) NOT NULL DEFAULT 'OPEN', -- OPEN / CLOSE
	remark varchar(255) NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT wx_mp_apps_pkey PRIMARY KEY (id)
);

COMMENT ON TABLE public.wx_mp_apps IS '微信小程序配置（按租户）';
COMMENT ON COLUMN public.wx_mp_apps.status IS '状态: OPEN/CLOSE';

CREATE UNIQUE INDEX IF NOT EXISTS uk_wx_mp_apps_tenant_id
	ON public.wx_mp_apps (tenant_id);

-- ============================================================================
-- 3) auth_message_templates：验证码/通知模板（按租户、按场景）
--    注意：供应商密钥等仍复用 notification_services_config（SYS_ADMIN配置）。
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.auth_message_templates (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	channel varchar(16) NOT NULL,  -- EMAIL / SMS
	scene varchar(32) NOT NULL,    -- LOGIN / REGISTER / RESET_PASSWORD / BIND
	subject varchar(200) NULL,     -- 邮件主题（SMS可空）
	content text NULL,             -- 内容模板（EMAIL可用；SMS可空）
	provider varchar(36) NULL,     -- 供应商（例如 ALIYUN；EMAIL可空）
	provider_template_code varchar(64) NULL, -- 供应商模板ID（短信多场景）
	status varchar(16) NOT NULL DEFAULT 'OPEN', -- OPEN / CLOSE
	remark varchar(255) NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT auth_message_templates_pkey PRIMARY KEY (id)
);

COMMENT ON TABLE public.auth_message_templates IS 'APP认证相关消息模板（验证码等，按租户/场景）';
COMMENT ON COLUMN public.auth_message_templates.channel IS '通道: EMAIL/SMS';
COMMENT ON COLUMN public.auth_message_templates.scene IS '场景: LOGIN/REGISTER/RESET_PASSWORD/BIND';
COMMENT ON COLUMN public.auth_message_templates.provider_template_code IS '短信供应商模板ID（多场景）';
COMMENT ON COLUMN public.auth_message_templates.status IS '状态: OPEN/CLOSE';

CREATE UNIQUE INDEX IF NOT EXISTS uk_auth_message_templates_tenant_channel_scene
	ON public.auth_message_templates (tenant_id, channel, scene);

-- ============================================================================
-- 4) 数据修正：历史WEB账号默认归类为 ORG_USER（避免被识别为 END_USER）
-- ============================================================================
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='user_kind') THEN
		UPDATE public.users SET user_kind = 'ORG_USER' WHERE user_kind IS NULL;
	END IF;
END $$;
