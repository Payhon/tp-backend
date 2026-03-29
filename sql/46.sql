-- Version: 46
-- Description: 清理后台删除用户后遗留的账号身份/绑定孤儿数据

-- 说明：
-- 历史上后台管理删除 users 记录时，未同步清理 APP 账号相关表，
-- 会导致 user_identities / device_user_bindings 等残留“孤儿数据”，
-- 进一步触发“手机号/邮箱/微信已被绑定，但后台又查不到主账号”的异常。

DELETE FROM public.user_identities ui
WHERE NOT EXISTS (
	SELECT 1
	FROM public.users u
	WHERE u.id = ui.user_id
);

DELETE FROM public.device_user_bindings dub
WHERE NOT EXISTS (
	SELECT 1
	FROM public.users u
	WHERE u.id = dub.user_id
);

DELETE FROM public.app_device_added_records ar
WHERE NOT EXISTS (
	SELECT 1
	FROM public.users u
	WHERE u.id = ar.user_id
);

DELETE FROM public.message_push_manage mpm
WHERE NOT EXISTS (
	SELECT 1
	FROM public.users u
	WHERE u.id = mpm.user_id
);

DELETE FROM public.message_push_log mpl
WHERE NOT EXISTS (
	SELECT 1
	FROM public.users u
	WHERE u.id = mpl.user_id
);

DELETE FROM public.user_roles ur
WHERE NOT EXISTS (
	SELECT 1
	FROM public.users u
	WHERE u.id = ur.user_id
);

DELETE FROM public.user_address ua
WHERE NOT EXISTS (
	SELECT 1
	FROM public.users u
	WHERE u.id = ua.user_id
);
