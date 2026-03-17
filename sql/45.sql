-- FEAT-0015: 后台用户体系与角色权限重构

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'is_main'
    ) THEN
        ALTER TABLE public.users ADD COLUMN is_main smallint NOT NULL DEFAULT 0;
        COMMENT ON COLUMN public.users.is_main IS '是否主账号 0-否 1-是';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'roles' AND column_name = 'authority'
    ) THEN
        ALTER TABLE public.roles ADD COLUMN authority varchar(50) NULL;
        COMMENT ON COLUMN public.roles.authority IS '角色适用账号权限类型';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'roles' AND column_name = 'user_kind'
    ) THEN
        ALTER TABLE public.roles ADD COLUMN user_kind varchar(50) NULL;
        COMMENT ON COLUMN public.roles.user_kind IS '角色适用用户类型';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'roles' AND column_name = 'org_type'
    ) THEN
        ALTER TABLE public.roles ADD COLUMN org_type varchar(50) NULL;
        COMMENT ON COLUMN public.roles.org_type IS '角色适用组织类型';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS public.user_roles (
    id varchar(36) NOT NULL,
    tenant_id varchar(36) NOT NULL,
    user_id varchar(36) NOT NULL,
    role_id varchar(36) NOT NULL,
    created_at timestamptz(6) NULL,
    updated_at timestamptz(6) NULL,
    CONSTRAINT user_roles_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE public.user_roles IS '用户角色关联表';
COMMENT ON COLUMN public.user_roles.tenant_id IS '租户ID';
COMMENT ON COLUMN public.user_roles.user_id IS '用户ID';
COMMENT ON COLUMN public.user_roles.role_id IS '角色ID';
COMMENT ON COLUMN public.user_roles.created_at IS '创建时间';
COMMENT ON COLUMN public.user_roles.updated_at IS '更新时间';

CREATE TABLE IF NOT EXISTS public.role_permissions (
    id varchar(36) NOT NULL,
    tenant_id varchar(36) NOT NULL,
    role_id varchar(36) NOT NULL,
    permission_key varchar(128) NOT NULL,
    created_at timestamptz(6) NULL,
    updated_at timestamptz(6) NULL,
    CONSTRAINT role_permissions_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE public.role_permissions IS '角色权限关联表';
COMMENT ON COLUMN public.role_permissions.tenant_id IS '租户ID';
COMMENT ON COLUMN public.role_permissions.role_id IS '角色ID';
COMMENT ON COLUMN public.role_permissions.permission_key IS '权限标识（sys_ui_elements.id）';
COMMENT ON COLUMN public.role_permissions.created_at IS '创建时间';
COMMENT ON COLUMN public.role_permissions.updated_at IS '更新时间';

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_user_role
    ON public.user_roles (user_id, role_id);

CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_user
    ON public.user_roles (tenant_id, user_id);

CREATE INDEX IF NOT EXISTS idx_user_roles_role_id
    ON public.user_roles (role_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_role_permissions_role_permission
    ON public.role_permissions (role_id, permission_key);

CREATE INDEX IF NOT EXISTS idx_role_permissions_tenant_role
    ON public.role_permissions (tenant_id, role_id);

CREATE INDEX IF NOT EXISTS idx_users_tenant_authority_kind_main
    ON public.users (tenant_id, authority, user_kind, is_main);

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_tenant_main_account
    ON public.users (tenant_id)
    WHERE authority = 'TENANT_ADMIN' AND user_kind = 'ORG_USER' AND is_main = 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_org_main_account
    ON public.users (tenant_id, org_id)
    WHERE authority = 'TENANT_USER' AND user_kind = 'ORG_USER' AND is_main = 1 AND org_id IS NOT NULL;

-- 历史数据回填
UPDATE public.users
SET user_kind = 'ORG_USER'
WHERE authority = 'TENANT_ADMIN'
  AND (user_kind IS NULL OR btrim(user_kind) = '');

UPDATE public.users
SET is_main = 0
WHERE is_main IS DISTINCT FROM 0
  AND (user_kind = 'END_USER' OR authority = 'SYS_ADMIN');

WITH tenant_ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY tenant_id
            ORDER BY created_at ASC NULLS LAST, id ASC
        ) AS rn
    FROM public.users
    WHERE authority = 'TENANT_ADMIN'
      AND tenant_id IS NOT NULL
      AND tenant_id <> ''
      AND COALESCE(user_kind, 'ORG_USER') = 'ORG_USER'
)
UPDATE public.users u
SET is_main = CASE WHEN tr.rn = 1 THEN 1 ELSE 0 END
FROM tenant_ranked tr
WHERE u.id = tr.id;

WITH org_ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY tenant_id, org_id
            ORDER BY created_at ASC NULLS LAST, id ASC
        ) AS rn
    FROM public.users
    WHERE authority = 'TENANT_USER'
      AND tenant_id IS NOT NULL
      AND tenant_id <> ''
      AND org_id IS NOT NULL
      AND org_id <> ''
      AND COALESCE(user_kind, 'ORG_USER') = 'ORG_USER'
)
UPDATE public.users u
SET is_main = CASE WHEN org.rn = 1 THEN 1 ELSE 0 END
FROM org_ranked org
WHERE u.id = org.id;

UPDATE public.users
SET is_main = 0
WHERE user_kind = 'END_USER' AND is_main <> 0;

-- 菜单迁移到 系统管理 > 后台账号管理 / 后台角色管理
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM public.sys_ui_elements WHERE id = 'a61b4a30-9d22-4b11-9f11-1a2b3c4d5e61'
    ) THEN
        UPDATE public.sys_ui_elements
        SET
            parent_id = 'e1ebd134-53df-3105-35f4-489fc674d173',
            orders = 46,
            param1 = '/management/backoffice-user',
            description = '后台账号管理',
            multilingual = 'route.bms_system_user',
            route_path = 'view.bms_system_user'
        WHERE id = 'a61b4a30-9d22-4b11-9f11-1a2b3c4d5e61';
    ELSE
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description,
            created_at, remark, multilingual, route_path
        ) VALUES (
            'a61b4a30-9d22-4b11-9f11-1a2b3c4d5e61',
            'e1ebd134-53df-3105-35f4-489fc674d173',
            'bms_system_user',
            3,
            46,
            '/management/backoffice-user',
            'mdi:account-cog',
            'self',
            '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '后台账号管理',
            NOW(),
            '',
            'route.bms_system_user',
            'view.bms_system_user'
        );
    END IF;

    IF EXISTS (
        SELECT 1 FROM public.sys_ui_elements WHERE id = 'b72c5b41-8e33-4c22-8f22-2b3c4d5e6f72'
    ) THEN
        UPDATE public.sys_ui_elements
        SET
            parent_id = 'e1ebd134-53df-3105-35f4-489fc674d173',
            orders = 47,
            param1 = '/management/backoffice-role',
            description = '后台角色管理',
            multilingual = 'route.bms_system_role',
            route_path = 'view.bms_system_role'
        WHERE id = 'b72c5b41-8e33-4c22-8f22-2b3c4d5e6f72';
    ELSE
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description,
            created_at, remark, multilingual, route_path
        ) VALUES (
            'b72c5b41-8e33-4c22-8f22-2b3c4d5e6f72',
            'e1ebd134-53df-3105-35f4-489fc674d173',
            'bms_system_role',
            3,
            47,
            '/management/backoffice-role',
            'mdi:account-key',
            'self',
            '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '后台角色管理',
            NOW(),
            '',
            'route.bms_system_role',
            'view.bms_system_role'
        );
    END IF;

    DELETE FROM public.sys_ui_elements
    WHERE id = '9e6a0c3a-2b7f-4d8c-9f1a-2d3b4c5d6e70'
      AND NOT EXISTS (
        SELECT 1 FROM public.sys_ui_elements child
        WHERE child.parent_id = '9e6a0c3a-2b7f-4d8c-9f1a-2d3b4c5d6e70'
      );
END $$;
