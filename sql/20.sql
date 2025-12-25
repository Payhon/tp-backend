-- BMS Project Database Migration Script
-- Version: 20
-- Description: Org Tree 多层级组织改造（BMS厂/PACK厂/经销商/门店）

-- ============================================================================
-- 1. 组织表 (orgs) - 统一表达 BMS厂/PACK厂/经销商/门店
-- ============================================================================
CREATE TABLE IF NOT EXISTS orgs (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    org_type VARCHAR(20) NOT NULL, -- BMS_FACTORY, PACK_FACTORY, DEALER, STORE
    parent_id VARCHAR(36), -- 上级组织ID，顶级组织为NULL
    tenant_id VARCHAR(36) NOT NULL,
    contact_person VARCHAR(100),
    phone VARCHAR(50),
    email VARCHAR(100),
    province VARCHAR(50),
    city VARCHAR(50),
    district VARCHAR(50),
    address VARCHAR(255),
    status VARCHAR(10) DEFAULT 'N', -- N-正常, F-禁用
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    remark VARCHAR(255)
);

COMMENT ON TABLE orgs IS '组织表（统一表达BMS厂/PACK厂/经销商/门店）';
COMMENT ON COLUMN orgs.id IS '组织ID';
COMMENT ON COLUMN orgs.name IS '组织名称';
COMMENT ON COLUMN orgs.org_type IS '组织类型: BMS_FACTORY-BMS厂家, PACK_FACTORY-PACK厂家, DEALER-经销商, STORE-门店';
COMMENT ON COLUMN orgs.parent_id IS '上级组织ID';
COMMENT ON COLUMN orgs.tenant_id IS '租户ID';
COMMENT ON COLUMN orgs.status IS '状态: N-正常, F-禁用';

-- 索引
CREATE INDEX IF NOT EXISTS idx_orgs_tenant_id ON orgs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_orgs_parent_id ON orgs(parent_id);
CREATE INDEX IF NOT EXISTS idx_orgs_tenant_org_type ON orgs(tenant_id, org_type);

-- ============================================================================
-- 2. 组织闭包表 (org_closure) - 用于子树查询与权限过滤
-- ============================================================================
CREATE TABLE IF NOT EXISTS org_closure (
    tenant_id VARCHAR(36) NOT NULL,
    ancestor_id VARCHAR(36) NOT NULL, -- 祖先组织ID
    descendant_id VARCHAR(36) NOT NULL, -- 后代组织ID（包含自身）
    depth INT NOT NULL DEFAULT 0, -- 层级深度: 0=自身, 1=直接子节点, 2=孙节点...
    PRIMARY KEY (tenant_id, ancestor_id, descendant_id)
);

COMMENT ON TABLE org_closure IS '组织闭包表（存储祖先-后代关系，用于子树查询）';
COMMENT ON COLUMN org_closure.tenant_id IS '租户ID';
COMMENT ON COLUMN org_closure.ancestor_id IS '祖先组织ID';
COMMENT ON COLUMN org_closure.descendant_id IS '后代组织ID（包含自身）';
COMMENT ON COLUMN org_closure.depth IS '层级深度: 0=自身, 1=直接子节点, 2=孙节点...';

-- 索引（用于不同查询场景）
CREATE INDEX IF NOT EXISTS idx_org_closure_ancestor ON org_closure(tenant_id, ancestor_id);
CREATE INDEX IF NOT EXISTS idx_org_closure_descendant ON org_closure(tenant_id, descendant_id);

-- ============================================================================
-- 3. 设备组织转移记录表 (device_org_transfers) - 替代原 device_transfers 的 dealer 语义
-- ============================================================================
CREATE TABLE IF NOT EXISTS device_org_transfers (
    id VARCHAR(36) PRIMARY KEY,
    device_id VARCHAR(36) NOT NULL,
    from_org_id VARCHAR(36), -- NULL表示从厂家直接出货
    to_org_id VARCHAR(36), -- NULL表示退回厂家
    operator_id VARCHAR(36), -- 操作人ID
    transfer_time TIMESTAMPTZ DEFAULT NOW(),
    remark TEXT,
    tenant_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE device_org_transfers IS '设备组织转移记录表';
COMMENT ON COLUMN device_org_transfers.id IS '记录ID';
COMMENT ON COLUMN device_org_transfers.device_id IS '设备ID';
COMMENT ON COLUMN device_org_transfers.from_org_id IS '转出组织ID（NULL表示从厂家出货）';
COMMENT ON COLUMN device_org_transfers.to_org_id IS '转入组织ID（NULL表示退回厂家）';
COMMENT ON COLUMN device_org_transfers.operator_id IS '操作人ID';
COMMENT ON COLUMN device_org_transfers.tenant_id IS '租户ID';

-- 索引
CREATE INDEX IF NOT EXISTS idx_device_org_transfers_device_id ON device_org_transfers(device_id);
CREATE INDEX IF NOT EXISTS idx_device_org_transfers_tenant_id ON device_org_transfers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_device_org_transfers_time ON device_org_transfers(transfer_time DESC);

-- ============================================================================
-- 4. 为 users 表增加 org 相关字段
-- ============================================================================
DO $$
BEGIN
    -- 添加 org_id 字段（业务账号归属组织）
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='org_id') THEN
        ALTER TABLE users ADD COLUMN org_id VARCHAR(36);
        COMMENT ON COLUMN users.org_id IS '归属组织ID（业务账号归属的组织）';
    END IF;

    -- 添加 user_kind 字段（区分业务账号与终端用户）
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='user_kind') THEN
        ALTER TABLE users ADD COLUMN user_kind VARCHAR(20) DEFAULT 'END_USER';
        COMMENT ON COLUMN users.user_kind IS '用户类型: ORG_USER-组织用户（业务账号）, END_USER-终端用户';
    END IF;
END $$;

-- 索引
CREATE INDEX IF NOT EXISTS idx_users_org_id ON users(org_id);
CREATE INDEX IF NOT EXISTS idx_users_user_kind ON users(user_kind);

-- ============================================================================
-- 5. 为 device_batteries 表增加 org 相关字段
-- ============================================================================
DO $$
BEGIN
    -- 添加 owner_org_id 字段（当前持有方组织）
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='owner_org_id') THEN
        ALTER TABLE device_batteries ADD COLUMN owner_org_id VARCHAR(36);
        COMMENT ON COLUMN device_batteries.owner_org_id IS '当前持有方组织ID（PACK/经销商/门店/或BMS厂）';
    END IF;

    -- 添加 bms_factory_org_id 字段（BMS板卡出厂方，固定）
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='bms_factory_org_id') THEN
        ALTER TABLE device_batteries ADD COLUMN bms_factory_org_id VARCHAR(36);
        COMMENT ON COLUMN device_batteries.bms_factory_org_id IS 'BMS板卡出厂方组织ID（固定，用于溯源）';
    END IF;

    -- 添加 pack_factory_org_id 字段（PACK组装方，可空）
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='pack_factory_org_id') THEN
        ALTER TABLE device_batteries ADD COLUMN pack_factory_org_id VARCHAR(36);
        COMMENT ON COLUMN device_batteries.pack_factory_org_id IS 'PACK组装方组织ID（可空，用于溯源）';
    END IF;
END $$;

-- 索引
CREATE INDEX IF NOT EXISTS idx_device_batteries_owner_org_id ON device_batteries(owner_org_id);
CREATE INDEX IF NOT EXISTS idx_device_batteries_bms_factory_org_id ON device_batteries(bms_factory_org_id);
CREATE INDEX IF NOT EXISTS idx_device_batteries_pack_factory_org_id ON device_batteries(pack_factory_org_id);

-- ============================================================================
-- 6. 注册组织管理相关菜单
-- ============================================================================

-- 6.1 组织管理主菜单
INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
)
VALUES (
    'org-management-main',
    'a753c525-780f-415f-a2b6-3d909c79f7f6',
    'bms_org_management',
    3,
    1400,
    '/bms/org/management',
    'mdi:sitemap',
    'self',
    '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '组织管理',
    NOW(),
    '',
    'route.bms_org_management',
    'view.bms_org_management'
) ON CONFLICT (id) DO NOTHING;

-- 6.2 PACK厂家管理（快捷方式）
INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
)
VALUES (
    'org-pack-factory',
    'a753c525-780f-415f-a2b6-3d909c79f7f6',
    'bms_pack_factory',
    3,
    1401,
    '/bms/org/management?org_type=PACK_FACTORY',
    'mdi:factory',
    'self',
    '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    'PACK厂家管理',
    NOW(),
    '',
    'route.bms_pack_factory',
    'view.bms_pack_factory'
) ON CONFLICT (id) DO NOTHING;

-- 6.3 经销商管理（快捷方式）
INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
)
VALUES (
    'org-dealer',
    'a753c525-780f-415f-a2b6-3d909c79f7f6',
    'bms_dealer',
    3,
    1402,
    '/bms/org/management?org_type=DEALER',
    'mdi:store',
    'self',
    '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '经销商管理',
    NOW(),
    '',
    'route.bms_dealer',
    'view.bms_dealer'
) ON CONFLICT (id) DO NOTHING;

-- 6.4 门店管理（快捷方式）
INSERT INTO public.sys_ui_elements (
    id, parent_id, element_code, element_type, orders,
    param1, param2, param3, authority, description,
    created_at, remark, multilingual, route_path
)
VALUES (
    'org-store',
    'a753c525-780f-415f-a2b6-3d909c79f7f6',
    'bms_store',
    3,
    1403,
    '/bms/org/management?org_type=STORE',
    'mdi:storefront',
    'self',
    '["TENANT_ADMIN","SYS_ADMIN"]'::json,
    '门店管理',
    NOW(),
    '',
    'route.bms_store',
    'view.bms_store'
) ON CONFLICT (id) DO NOTHING;
