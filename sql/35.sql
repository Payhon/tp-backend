-- Version: 35
-- Description: sys_dict add tenant_id and category (tenant-aware dictionaries)

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='sys_dict' AND column_name='tenant_id'
  ) THEN
    ALTER TABLE public.sys_dict ADD COLUMN tenant_id varchar(36) NOT NULL DEFAULT '0';
    COMMENT ON COLUMN public.sys_dict.tenant_id IS '租户ID(0-系统全局)';
    CREATE INDEX IF NOT EXISTS idx_sys_dict_tenant_id ON public.sys_dict(tenant_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='sys_dict' AND column_name='category'
  ) THEN
    ALTER TABLE public.sys_dict ADD COLUMN category varchar(100) NOT NULL DEFAULT 'default';
    COMMENT ON COLUMN public.sys_dict.category IS '字典类别';
    CREATE INDEX IF NOT EXISTS idx_sys_dict_category ON public.sys_dict(category);
  END IF;

  -- ensure historical rows have defaults
  UPDATE public.sys_dict SET tenant_id='0' WHERE tenant_id IS NULL OR tenant_id='';
  UPDATE public.sys_dict SET category='default' WHERE category IS NULL OR category='';

  -- update unique constraint: tenant_id + dict_code + dict_value
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='sys_dict_dict_code_dict_value_key'
  ) THEN
    ALTER TABLE public.sys_dict DROP CONSTRAINT sys_dict_dict_code_dict_value_key;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='sys_dict_tenant_dict_code_dict_value_key'
  ) THEN
    ALTER TABLE public.sys_dict
      ADD CONSTRAINT sys_dict_tenant_dict_code_dict_value_key UNIQUE (tenant_id, dict_code, dict_value);
    COMMENT ON CONSTRAINT sys_dict_tenant_dict_code_dict_value_key ON public.sys_dict IS 'tenant_id+dict_code+dict_value唯一';
  END IF;
END $$;

