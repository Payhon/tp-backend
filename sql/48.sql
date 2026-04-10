-- FEAT-0037: users.username 字段、历史用户名回填与唯一索引

ALTER TABLE public.users ADD COLUMN IF NOT EXISTS username varchar(255) NULL;
COMMENT ON COLUMN public.users.username IS '账号用户名';

DO $$
DECLARE
	rec record;
	base_username varchar(255);
	attempt_username varchar(255);
	suffix varchar(6);
	counter integer;
BEGIN
	FOR rec IN
		SELECT id, tenant_id, user_kind, phone_number, email
		FROM public.users
		WHERE username IS NULL OR btrim(username) = ''
		ORDER BY created_at NULLS LAST, id
	LOOP
		base_username := NULL;

		IF COALESCE(rec.user_kind, 'END_USER') = 'END_USER' THEN
			IF btrim(COALESCE(rec.phone_number, '')) <> '' THEN
				base_username := btrim(rec.phone_number);
			ELSIF btrim(COALESCE(rec.email, '')) <> ''
				AND lower(btrim(rec.email)) !~ '^(u|org)_[0-9a-f]+@app\.local$' THEN
				base_username := btrim(rec.email);
			END IF;
		ELSE
			IF btrim(COALESCE(rec.email, '')) <> ''
				AND lower(btrim(rec.email)) !~ '^(u|org)_[0-9a-f]+@app\.local$' THEN
				base_username := btrim(rec.email);
			ELSIF btrim(COALESCE(rec.phone_number, '')) <> '' THEN
				base_username := btrim(rec.phone_number);
			END IF;
		END IF;

		IF base_username IS NULL OR base_username = '' THEN
			CONTINUE;
		END IF;

		attempt_username := left(base_username, 255);
		IF EXISTS (
			SELECT 1
			FROM public.users u2
			WHERE u2.tenant_id = rec.tenant_id
				AND u2.id <> rec.id
				AND u2.username IS NOT NULL
				AND btrim(u2.username) <> ''
				AND lower(u2.username) = lower(attempt_username)
		) THEN
			suffix := right(replace(rec.id, '-', ''), 6);
			attempt_username := left(base_username, 248) || '_' || suffix;
			counter := 2;

			WHILE EXISTS (
				SELECT 1
				FROM public.users u2
				WHERE u2.tenant_id = rec.tenant_id
					AND u2.id <> rec.id
					AND u2.username IS NOT NULL
					AND btrim(u2.username) <> ''
					AND lower(u2.username) = lower(attempt_username)
			) LOOP
				attempt_username := left(base_username, GREATEST(1, 248 - length(counter::text) - 1))
					|| '_' || suffix || '_' || counter::text;
				counter := counter + 1;
			END LOOP;
		END IF;

		UPDATE public.users
		SET username = attempt_username
		WHERE id = rec.id
			AND (username IS NULL OR btrim(username) = '');
	END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_tenant_username_ci
	ON public.users(tenant_id, LOWER(username))
	WHERE username IS NOT NULL AND username <> '';
