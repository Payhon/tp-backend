-- ✅2025/12/25 WEB：新增 APP 认证配置菜单（模板/微信小程序配置）

DO $$
BEGIN
	-- 系统管理 -> APP认证配置
	IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'management_app-auth-config') THEN
		INSERT INTO public.sys_ui_elements (
			id, parent_id, element_code, element_type, orders,
			param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
		) VALUES (
			'1f3c5159-1c5a-4580-8a56-10a0c9f8e82a',
			'e1ebd134-53df-3105-35f4-489fc674d173',
			'management_app-auth-config',
			3,
			45,
			'/management/app-auth-config',
			'mdi:account-key',
			'self',
			'["SYS_ADMIN","TENANT_ADMIN"]'::json,
			'APP认证配置',
			NOW(),
			'',
			'route.management_app-auth-config',
			'view.management_app-auth-config'
		);
	END IF;
END $$;

