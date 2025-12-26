-- ✅2025/12/26 WEB：机构类型菜单权限字段改为 element_code（menu_ids -> ui_codes）

DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'org_type_permissions') THEN
		-- 旧表结构：menu_ids(jsonb) -> 新结构：ui_codes(jsonb)
		IF EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'org_type_permissions' AND column_name = 'menu_ids'
		) AND NOT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'org_type_permissions' AND column_name = 'ui_codes'
		) THEN
			ALTER TABLE public.org_type_permissions ADD COLUMN ui_codes jsonb NOT NULL DEFAULT '[]'::jsonb;

			-- 将旧的 sys_ui_elements.id 列表转换为 sys_ui_elements.element_code 列表
			UPDATE public.org_type_permissions otp
			SET ui_codes = COALESCE(
				(
					SELECT jsonb_agg(e.element_code)
					FROM public.sys_ui_elements e
					WHERE e.id IN (SELECT jsonb_array_elements_text(otp.menu_ids))
				),
				'[]'::jsonb
			);

			ALTER TABLE public.org_type_permissions DROP COLUMN menu_ids;

			COMMENT ON COLUMN public.org_type_permissions.ui_codes IS '菜单权限（sys_ui_elements.element_code JSON数组）';
		END IF;
	END IF;
END $$;

