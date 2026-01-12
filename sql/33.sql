-- Version: 33
-- Description: device_batteries add product_spec/order_number (required by battery ops)

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='product_spec') THEN
        ALTER TABLE device_batteries ADD COLUMN product_spec VARCHAR(32);
        COMMENT ON COLUMN device_batteries.product_spec IS '产品规格';
        CREATE INDEX IF NOT EXISTS idx_device_batteries_product_spec ON device_batteries(product_spec);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='device_batteries' AND column_name='order_number') THEN
        ALTER TABLE device_batteries ADD COLUMN order_number VARCHAR(32);
        COMMENT ON COLUMN device_batteries.order_number IS '订单编号';
        CREATE INDEX IF NOT EXISTS idx_device_batteries_order_number ON device_batteries(order_number);
    END IF;
END $$;

