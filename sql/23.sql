-- ✅2025/12/25 WEB：新增机构类型权限配置（菜单权限/设备参数权限）+ 权限管理菜单

-- ============================================================================
-- 1) 机构类型权限配置表（按租户 + 机构类型）
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.org_type_permissions (
	tenant_id varchar(36) NOT NULL,
	org_type varchar(20) NOT NULL, -- PACK_FACTORY / DEALER / STORE
	ui_codes jsonb NOT NULL DEFAULT '[]'::jsonb, -- sys_ui_elements.element_code 列表
	device_param_permissions text NULL, -- 逗号分割字符串
	created_at timestamptz(6) NOT NULL DEFAULT now(),
	updated_at timestamptz(6) NOT NULL DEFAULT now(),
	CONSTRAINT org_type_permissions_pk PRIMARY KEY (tenant_id, org_type)
);

COMMENT ON TABLE public.org_type_permissions IS '机构类型权限配置（菜单权限/设备参数权限）';
COMMENT ON COLUMN public.org_type_permissions.tenant_id IS '租户ID';
COMMENT ON COLUMN public.org_type_permissions.org_type IS '机构类型（PACK_FACTORY/DEALER/STORE）';
COMMENT ON COLUMN public.org_type_permissions.ui_codes IS '菜单权限（sys_ui_elements.element_code JSON数组）';
COMMENT ON COLUMN public.org_type_permissions.device_param_permissions IS '设备参数权限（逗号分割字符串）';

CREATE INDEX IF NOT EXISTS idx_org_type_permissions_tenant ON public.org_type_permissions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_org_type_permissions_org_type ON public.org_type_permissions(org_type);

-- ============================================================================
-- 2) 系统管理 -> 权限管理 菜单
-- ============================================================================
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'management_permission') THEN
		INSERT INTO public.sys_ui_elements (
			id, parent_id, element_code, element_type, orders,
			param1, param2, param3, authority, description, created_at, remark, multilingual, route_path
		) VALUES (
			'9b9c7d6e-8b4a-4f01-9b7f-9f6f6a2c2f33',
			'e1ebd134-53df-3105-35f4-489fc674d173',
			'management_permission',
			3,
			46,
			'/management/permission',
			'mdi:shield-key',
			'self',
			'["SYS_ADMIN","TENANT_ADMIN"]'::json,
			'权限管理',
			NOW(),
			'',
			'route.management_permission',
			'view.management_permission'
		);
	END IF;
END $$;
