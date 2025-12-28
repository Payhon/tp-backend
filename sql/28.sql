-- Version: 28
-- Description: 文件存储抽象（配置 + 文件表）

-- ============================================================================
-- 1. 文件存储配置（系统级）
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.file_storage_config (
	id varchar(36) NOT NULL,
	storage_type varchar(10) NOT NULL DEFAULT 'local', -- local/cloud
	provider varchar(10) NULL, -- aliyun/qiniu（storage_type=cloud时生效）
	config jsonb NOT NULL DEFAULT '{}'::jsonb, -- 供应商/本地配置（包含域名、bucket、AK/SK 等）
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	remark varchar(255) NULL,
	CONSTRAINT file_storage_config_pkey PRIMARY KEY (id)
);

COMMENT ON TABLE public.file_storage_config IS '文件存储配置（系统设置：本地/云存储 + 供应商配置）';
COMMENT ON COLUMN public.file_storage_config.storage_type IS '存储类型：local/cloud';
COMMENT ON COLUMN public.file_storage_config.provider IS '云存储供应商：aliyun/qiniu';
COMMENT ON COLUMN public.file_storage_config.config IS '配置（jsonb）：包含域名、bucket、AK/SK 等';

INSERT INTO public.file_storage_config (id, storage_type, provider, config)
SELECT 'file_storage_config_1', 'local', NULL, '{}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM public.file_storage_config WHERE id = 'file_storage_config_1');

-- ============================================================================
-- 2. 文件表（记录上传信息）
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.files (
	id varchar(36) NOT NULL,
	tenant_id varchar(36) NOT NULL, -- 租户ID
	file_name varchar(255) NOT NULL, -- 文件名（用于展示）
	original_file_name varchar(255) NULL, -- 原始文件名（上传时）
	file_size int8 NOT NULL, -- 文件大小（字节）
	storage_location varchar(10) NOT NULL, -- local/aliyun/qiniu
	biz_type varchar(50) NOT NULL, -- 业务类型（上传时传入 type）
	mime_type varchar(100) NULL, -- 文件类型（MIME）
	file_ext varchar(20) NULL, -- 文件扩展名（.png）
	md5 varchar(32) NULL,
	sha256 varchar(64) NULL,
	file_path varchar(500) NOT NULL, -- 文件路径（local为相对路径，云为object key）
	full_url varchar(1000) NOT NULL, -- 可访问URL（云为域名+key，本地为服务地址+files）
	uploaded_at timestamptz NOT NULL DEFAULT NOW(), -- 上传时间
	uploaded_by varchar(36) NULL, -- 上传用户ID
	meta jsonb NOT NULL DEFAULT '{}'::jsonb, -- 额外信息
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	remark varchar(255) NULL,
	CONSTRAINT files_pkey PRIMARY KEY (id),
	CONSTRAINT files_users_fk FOREIGN KEY (uploaded_by) REFERENCES public.users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

COMMENT ON TABLE public.files IS '文件表：记录文件上传信息';
COMMENT ON COLUMN public.files.tenant_id IS '租户ID';
COMMENT ON COLUMN public.files.file_name IS '文件名（用于展示）';
COMMENT ON COLUMN public.files.storage_location IS '存储位置：local/aliyun/qiniu';
COMMENT ON COLUMN public.files.biz_type IS '业务类型（上传时传入 type）';
COMMENT ON COLUMN public.files.file_path IS '文件路径：本地为相对路径，云为object key';
COMMENT ON COLUMN public.files.full_url IS '可访问URL';

CREATE INDEX IF NOT EXISTS idx_files_tenant_time ON public.files (tenant_id, uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_tenant_biz ON public.files (tenant_id, biz_type);
CREATE INDEX IF NOT EXISTS idx_files_storage_location ON public.files (storage_location);

