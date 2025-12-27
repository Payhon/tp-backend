-- Version: 26
-- Description: Fix APP管理菜单层级（确保顶层 route key 无下划线，避免前端被识别为非一级路由导致无外层布局）

DO $$
DECLARE
	root_id varchar(36);
BEGIN
	-- 如果旧的 app_manage 已存在，改名为 app 并作为顶层菜单
	SELECT id INTO root_id FROM public.sys_ui_elements WHERE element_code = 'app' LIMIT 1;

	IF root_id IS NULL THEN
		SELECT id INTO root_id FROM public.sys_ui_elements WHERE element_code = 'app_manage' LIMIT 1;
		IF root_id IS NOT NULL THEN
			UPDATE public.sys_ui_elements
			SET element_code = 'app',
				parent_id = '0',
				element_type = 1,
				param1 = '/app',
				param2 = COALESCE(param2, 'mdi:cellphone-cog'),
				param3 = COALESCE(param3, '0'),
				description = COALESCE(description, 'APP管理'),
				multilingual = COALESCE(multilingual, 'route.app_manage'),
				route_path = COALESCE(route_path, 'layout.base')
			WHERE id = root_id;
		END IF;
	END IF;

	-- 如果还没有 root，则创建
	IF root_id IS NULL THEN
		root_id := 'b9c8f7b5-6ef0-4c1b-9b6f-6c2c3a5a5c25';
		IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE id = root_id) THEN
			INSERT INTO public.sys_ui_elements (
				id, parent_id, element_code, element_type, orders,
				param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
			) VALUES (
				root_id, '0', 'app', 1, 80,
				'/app', 'mdi:cellphone-cog', '0', '["TENANT_ADMIN","SYS_ADMIN"]'::json,
				'APP管理', NOW(), '', 'route.app_manage', 'layout.base'
			);
		END IF;
	END IF;

	-- 子菜单统一挂到 app 顶层下，并修正路径
	UPDATE public.sys_ui_elements SET parent_id = root_id, param1 = '/app/manage/apps'
	WHERE element_code = 'app_manage_apps';

	UPDATE public.sys_ui_elements SET parent_id = root_id, param1 = '/app/manage/upgrade'
	WHERE element_code = 'app_manage_upgrade';

	UPDATE public.sys_ui_elements SET parent_id = root_id, param1 = '/app/manage/users'
	WHERE element_code = 'app_manage_users';

	-- 避免旧的 app_manage 作为顶层残留（若存在且不是 root_id，则隐藏）
	UPDATE public.sys_ui_elements SET param3 = '1'
	WHERE element_code = 'app_manage' AND id <> root_id;
END $$;

