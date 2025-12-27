-- Version: 27
-- Description: APP内容管理（单页/FAQ/用户反馈）+ 菜单

-- ============================================================================
-- 1. 单页内容（用户政策/隐私政策...）
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.app_content_pages (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	app_id varchar(36) NOT NULL,
	content_key varchar(50) NOT NULL, -- user_policy / privacy_policy
	published bool NOT NULL DEFAULT false,
	published_at timestamptz NULL,
	created_by varchar(36) NULL,
	updated_by varchar(36) NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	remark text NULL,
	CONSTRAINT app_content_pages_pkey PRIMARY KEY (id),
	CONSTRAINT app_content_pages_apps_fk FOREIGN KEY (app_id) REFERENCES public.apps(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE public.app_content_pages IS 'APP内容管理：单页内容（按应用/Key）';
COMMENT ON COLUMN public.app_content_pages.content_key IS '内容Key：user_policy/privacy_policy';

CREATE UNIQUE INDEX IF NOT EXISTS uk_app_content_pages_tenant_app_key
	ON public.app_content_pages (tenant_id, app_id, content_key);
CREATE INDEX IF NOT EXISTS idx_app_content_pages_tenant_app
	ON public.app_content_pages (tenant_id, app_id);

CREATE TABLE IF NOT EXISTS public.app_content_page_i18n (
	id varchar(36) NOT NULL,
	page_id varchar(36) NOT NULL,
	lang varchar(10) NOT NULL, -- zh-CN / en-US
	title varchar(255) NULL,
	content_markdown text NULL,
	content_html text NULL,
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT app_content_page_i18n_pkey PRIMARY KEY (id),
	CONSTRAINT app_content_page_i18n_page_fk FOREIGN KEY (page_id) REFERENCES public.app_content_pages(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE public.app_content_page_i18n IS 'APP内容管理：单页内容多语言';

CREATE UNIQUE INDEX IF NOT EXISTS uk_app_content_page_i18n_page_lang
	ON public.app_content_page_i18n (page_id, lang);

-- ============================================================================
-- 2. FAQ（置顶+排序）
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.app_faq (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	app_id varchar(36) NOT NULL,
	is_pinned bool NOT NULL DEFAULT false,
	sort int4 NOT NULL DEFAULT 0, -- 数字越大越靠前
	published bool NOT NULL DEFAULT false,
	published_at timestamptz NULL,
	created_by varchar(36) NULL,
	updated_by varchar(36) NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	remark text NULL,
	CONSTRAINT app_faq_pkey PRIMARY KEY (id),
	CONSTRAINT app_faq_apps_fk FOREIGN KEY (app_id) REFERENCES public.apps(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE public.app_faq IS 'APP内容管理：FAQ（按应用）';

CREATE INDEX IF NOT EXISTS idx_app_faq_tenant_app_published
	ON public.app_faq (tenant_id, app_id, published, is_pinned DESC, sort DESC, updated_at DESC);

CREATE TABLE IF NOT EXISTS public.app_faq_i18n (
	id varchar(36) NOT NULL,
	faq_id varchar(36) NOT NULL,
	lang varchar(10) NOT NULL, -- zh-CN / en-US
	question varchar(500) NULL,
	answer_markdown text NULL,
	answer_html text NULL,
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT app_faq_i18n_pkey PRIMARY KEY (id),
	CONSTRAINT app_faq_i18n_faq_fk FOREIGN KEY (faq_id) REFERENCES public.app_faq(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE public.app_faq_i18n IS 'APP内容管理：FAQ多语言';

CREATE UNIQUE INDEX IF NOT EXISTS uk_app_faq_i18n_faq_lang
	ON public.app_faq_i18n (faq_id, lang);

-- ============================================================================
-- 3. 用户反馈（需登录提交，管理员回复可见）
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.app_feedback (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL,
	app_id varchar(36) NOT NULL,
	appid varchar(100) NOT NULL,
	user_id varchar(36) NULL,
	content text NOT NULL,
	images jsonb NOT NULL DEFAULT '[]'::jsonb,
	platform varchar(30) NULL,
	app_version varchar(50) NULL,
	device_model varchar(100) NULL,
	os_version varchar(50) NULL,
	status varchar(20) NOT NULL DEFAULT 'NEW', -- NEW/PROCESSING/RESOLVED/CLOSED
	reply text NULL, -- 管理员回复（App端可见）
	replied_at timestamptz NULL,
	handler_uid varchar(36) NULL,
	handle_note text NULL, -- 内部备注
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	CONSTRAINT app_feedback_pkey PRIMARY KEY (id),
	CONSTRAINT app_feedback_apps_fk FOREIGN KEY (app_id) REFERENCES public.apps(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT app_feedback_users_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

COMMENT ON TABLE public.app_feedback IS 'APP内容管理：用户反馈';

CREATE INDEX IF NOT EXISTS idx_app_feedback_tenant_app_status_time
	ON public.app_feedback (tenant_id, app_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_feedback_user_time
	ON public.app_feedback (user_id, created_at DESC);

-- ============================================================================
-- 4. 菜单：APP内容管理（sys_ui_elements）
-- ============================================================================
DO $$
DECLARE
	root_id varchar(36);
BEGIN
	SELECT id INTO root_id FROM public.sys_ui_elements WHERE element_code = 'app' LIMIT 1;
	IF root_id IS NULL THEN
		root_id := 'b9c8f7b5-6ef0-4c1b-9b6f-6c2c3a5a5c25';
	END IF;

	IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'app_manage_content') THEN
		INSERT INTO public.sys_ui_elements (
			id, parent_id, element_code, element_type, orders,
			param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
		) VALUES (
			'9f21e25b-23d1-4f1d-a2e1-1d5f8a39f3b7', root_id, 'app_manage_content', 3, 804,
			'/app/manage/content', 'mdi:file-document-edit-outline', '0', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
			'内容管理', NOW(), '', 'route.app_manage_content', 'view.app_manage_content'
		);
	END IF;
END $$;

