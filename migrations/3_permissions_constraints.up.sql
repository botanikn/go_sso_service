-- Keep a single permission row per (user, app): the most recent one wins.
DELETE FROM permissions p
USING permissions newer
WHERE p.user_id = newer.user_id
  AND p.app_id = newer.app_id
  AND p.id < newer.id;

DELETE FROM permissions WHERE user_id IS NULL OR app_id IS NULL;

-- 1_init may already define this constraint on fresh databases.
ALTER TABLE permissions DROP CONSTRAINT IF EXISTS unique_user_app;

ALTER TABLE permissions
    ALTER COLUMN user_id SET NOT NULL,
    ALTER COLUMN app_id SET NOT NULL,
    ADD CONSTRAINT permissions_user_app_unique UNIQUE (user_id, app_id);

-- UNIQUE on users.email already creates an index.
DROP INDEX IF EXISTS idx_email;

-- Match int64 IDs used by the service.
ALTER SEQUENCE users_id_seq AS BIGINT;
ALTER SEQUENCE apps_id_seq AS BIGINT;
ALTER SEQUENCE permissions_id_seq AS BIGINT;
ALTER TABLE users ALTER COLUMN id TYPE BIGINT;
ALTER TABLE apps ALTER COLUMN id TYPE BIGINT;
ALTER TABLE permissions
    ALTER COLUMN id TYPE BIGINT,
    ALTER COLUMN user_id TYPE BIGINT,
    ALTER COLUMN app_id TYPE BIGINT;
