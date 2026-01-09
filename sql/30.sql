-- Alarm: suggestions + processing fields

ALTER TABLE public.alarm_config
	ADD COLUMN IF NOT EXISTS processing_suggestions text NULL;

COMMENT ON COLUMN public.alarm_config.processing_suggestions IS '建议处理方式（换行分条）';

ALTER TABLE public.alarm_history
	ADD COLUMN IF NOT EXISTS processing_remark text NULL,
	ADD COLUMN IF NOT EXISTS processed_at timestamptz(6) NULL,
	ADD COLUMN IF NOT EXISTS processed_by varchar(36) NULL;

COMMENT ON COLUMN public.alarm_history.processing_remark IS '处理备注（App端手动处理填写）';
COMMENT ON COLUMN public.alarm_history.processed_at IS '处理时间（App端手动处理）';
COMMENT ON COLUMN public.alarm_history.processed_by IS '处理人（用户ID）';

