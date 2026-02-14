-- ✅2026/02/13 第三方 MES 对接：OpenAPI 密钥扩展 + 菜单补齐

-- 1) open_api_keys 扩展字段
ALTER TABLE public.open_api_keys
	ADD COLUMN IF NOT EXISTS secret_key varchar(255) NULL,
	ADD COLUMN IF NOT EXISTS remark varchar(255) NULL,
	ADD COLUMN IF NOT EXISTS expired_at timestamptz(6) NULL,
	ADD COLUMN IF NOT EXISTS last_used_at timestamptz(6) NULL;

COMMENT ON COLUMN public.open_api_keys.api_key IS '第三方调用 AppId';
COMMENT ON COLUMN public.open_api_keys.secret_key IS '第三方调用 SecretKey';
COMMENT ON COLUMN public.open_api_keys.remark IS '备注';
COMMENT ON COLUMN public.open_api_keys.expired_at IS '有效期截止时间';
COMMENT ON COLUMN public.open_api_keys.last_used_at IS '最近使用时间';

-- 历史数据回填（兼容旧版本仅有 api_key 的数据）
UPDATE public.open_api_keys
SET
	secret_key = COALESCE(NULLIF(secret_key, ''), 'sk_' || md5(random()::text || clock_timestamp()::text)),
	remark = COALESCE(NULLIF(remark, ''), NULLIF(name, ''), '默认密钥'),
	name = COALESCE(NULLIF(name, ''), COALESCE(NULLIF(remark, ''), '默认密钥'))
WHERE secret_key IS NULL OR secret_key = '' OR remark IS NULL OR remark = '' OR name IS NULL OR name = '';

CREATE INDEX IF NOT EXISTS idx_open_api_keys_tenant_status_expired
	ON public.open_api_keys (tenant_id, status, expired_at);

-- 2) 菜单：系统管理 -> API 密钥管理（保证 SYS_ADMIN/TENANT_ADMIN 可见）
UPDATE public.sys_ui_elements
SET
	param1 = '/management/api',
	param2 = 'mdi:key-chain-variant',
	param3 = '0',
	authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
	description = 'API密钥管理',
	multilingual = 'route.management_api',
	route_path = 'view.management_api'
WHERE element_code = 'management_api';

INSERT INTO public.sys_ui_elements
(id, parent_id, element_code, element_type, orders, param1, param2, param3, authority, description, created_at, remark, multilingual, route_path)
SELECT
	'9a16cc20-3e2e-4c5e-b5f3-1cfa2ea71127',
	'e1ebd134-53df-3105-35f4-489fc674d173',
	'management_api',
	3,
	1999,
	'/management/api',
	'mdi:key-chain-variant',
	'0',
	'["TENANT_ADMIN","SYS_ADMIN"]'::json,
	'API密钥管理',
	now(),
	'',
	'route.management_api',
	'view.management_api'
WHERE NOT EXISTS (
	SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'management_api'
);
