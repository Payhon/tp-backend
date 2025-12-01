-- BMS Project Database Migration Script
-- Version: 13
-- Description: Add tables for Battery Management System (Dealers, Battery Models, Lifecycle, Warranty)

-- 1. Dealers Table (经销商表)
CREATE TABLE IF NOT EXISTS dealers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    contact_person VARCHAR(100),
    phone VARCHAR(50),
    email VARCHAR(100),
    province VARCHAR(50),
    city VARCHAR(50),
    district VARCHAR(50),
    address VARCHAR(255),
    parent_id VARCHAR(36), -- For multi-level dealers
    tenant_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    remark VARCHAR(255)
);

COMMENT ON TABLE dealers IS '经销商表';
COMMENT ON COLUMN dealers.id IS '经销商ID';
COMMENT ON COLUMN dealers.name IS '经销商名称';
COMMENT ON COLUMN dealers.tenant_id IS '租户ID';

-- 2. Battery Models Table (电池型号表)
CREATE TABLE IF NOT EXISTS battery_models (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    voltage_rated FLOAT, -- Rated Voltage (V)
    capacity_rated FLOAT, -- Rated Capacity (Ah)
    cell_count INT, -- Number of cells
    nominal_power FLOAT, -- Nominal Power (W)
    warranty_months INT, -- Default warranty period in months
    description TEXT,
    tenant_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE battery_models IS '电池型号表';

-- 3. Device Batteries Extension Table (设备电池扩展信息表)
-- One-to-one relationship with devices table
CREATE TABLE IF NOT EXISTS device_batteries (
    device_id VARCHAR(36) PRIMARY KEY,
    battery_model_id VARCHAR(36),
    dealer_id VARCHAR(36), -- Current dealer ownership
    production_date DATE,
    warranty_expire_date DATE,
    activation_date TIMESTAMPTZ,
    activation_status VARCHAR(20) DEFAULT 'INACTIVE', -- INACTIVE, ACTIVE
    transfer_status VARCHAR(20) DEFAULT 'FACTORY', -- FACTORY, DEALER, USER
    batch_number VARCHAR(100),
    soc FLOAT, -- Last reported SOC
    soh FLOAT, -- Last reported SOH
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE device_batteries IS '设备电池扩展信息表';
COMMENT ON COLUMN device_batteries.activation_status IS '激活状态: INACTIVE, ACTIVE';
COMMENT ON COLUMN device_batteries.transfer_status IS '流转状态: FACTORY, DEALER, USER';

-- 4. Device Transfer Records (设备转移记录表)
CREATE TABLE IF NOT EXISTS device_transfers (
    id VARCHAR(36) PRIMARY KEY,
    device_id VARCHAR(36) NOT NULL,
    from_dealer_id VARCHAR(36), -- NULL means from Manufacturer
    to_dealer_id VARCHAR(36),
    operator_id VARCHAR(36),
    transfer_time TIMESTAMPTZ DEFAULT NOW(),
    remark TEXT,
    tenant_id VARCHAR(36) NOT NULL
);

COMMENT ON TABLE device_transfers IS '设备转移记录表';

-- 5. Device User Bindings (终端用户绑定关系表)
CREATE TABLE IF NOT EXISTS device_user_bindings (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    device_id VARCHAR(36) NOT NULL,
    binding_time TIMESTAMPTZ DEFAULT NOW(),
    is_owner BOOLEAN DEFAULT FALSE, -- Is primary owner
    remark VARCHAR(255)
);

COMMENT ON TABLE device_user_bindings IS '终端用户绑定关系表';
CREATE INDEX idx_device_user_bindings_user_id ON device_user_bindings(user_id);
CREATE INDEX idx_device_user_bindings_device_id ON device_user_bindings(device_id);

-- 6. Warranty Applications (维保申请表)
CREATE TABLE IF NOT EXISTS warranty_applications (
    id VARCHAR(36) PRIMARY KEY,
    device_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL, -- Applicant
    type VARCHAR(20), -- REPAIR, RETURN, EXCHANGE
    description TEXT,
    images JSONB, -- Array of image URLs
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, APPROVED, REJECTED, PROCESSING, COMPLETED
    result_info JSONB, -- Repair result or rejection reason
    handler_id VARCHAR(36), -- Operator who handled it
    tenant_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE warranty_applications IS '维保申请表';

-- 7. Add dealer_id to users table
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='dealer_id') THEN
        ALTER TABLE users ADD COLUMN dealer_id VARCHAR(36);
        COMMENT ON COLUMN users.dealer_id IS '归属经销商ID';
    END IF;
END $$;

-- 8. Add indexes for performance
CREATE INDEX idx_dealers_tenant_id ON dealers(tenant_id);
CREATE INDEX idx_device_batteries_dealer_id ON device_batteries(dealer_id);
CREATE INDEX idx_device_transfers_device_id ON device_transfers(device_id);
CREATE INDEX idx_warranty_applications_device_id ON warranty_applications(device_id);
