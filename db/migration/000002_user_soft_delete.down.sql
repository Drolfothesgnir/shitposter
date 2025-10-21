DROP VIEW IF EXISTS posts_with_author;

DROP INDEX IF EXISTS uniq_users_email_active;
DROP INDEX IF EXISTS uniq_users_username_active;

ALTER TABLE users
  ADD CONSTRAINT users_email_key UNIQUE (email);

ALTER TABLE users
  ADD CONSTRAINT users_username_key UNIQUE (username);

ALTER TABLE users
  DROP COLUMN IF EXISTS is_deleted,
  DROP COLUMN IF EXISTS deleted_at,
  DROP COLUMN IF EXISTS display_name,
  DROP COLUMN IF EXISTS archived_username,
  DROP COLUMN IF EXISTS archived_email;
