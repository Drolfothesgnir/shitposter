-- adding fields to allow soft delete
ALTER TABLE users
  ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN deleted_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  ADD COLUMN display_name VARCHAR NOT NULL DEFAULT '';

-- we need to drop unique email/username constraints to make those
-- emails and usenames available to 'live' users
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;

CREATE UNIQUE INDEX uniq_users_email_active
  ON users(email)
  WHERE is_deleted = false;

CREATE UNIQUE INDEX uniq_users_username_active
  ON users(username)
  WHERE is_deleted = false;

-- adding default display_name to all users
UPDATE users
  SET display_name = username
  WHERE display_name = '';

-- helper view for usage in differently-ordered post extraction
CREATE OR REPLACE VIEW posts_with_author AS
SELECT 
  p.id,
  p.user_id,
  p.title,
  p.topics,
  p.body,
  p.upvotes,
  p.downvotes,
  p.popularity,
  p.created_at,
  p.last_modified_at,
  u.display_name      AS user_display_name,
  u.profile_img_url   AS user_profile_img_url,
  u.is_deleted        AS user_is_deleted
FROM posts AS p
JOIN users AS u ON u.id = p.user_id;
