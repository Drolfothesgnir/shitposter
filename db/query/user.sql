-- name: CreateUser :one
INSERT INTO users (
  username, 
  display_name,
  profile_img_url,
  email,
  webauthn_user_handle
) VALUES (
  $1, $1, $2, $3, $4
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: UsernameExists :one 
SELECT EXISTS (SELECT 1 from users WHERE username = $1) AS username_exists;

-- name: EmailExists :one 
SELECT EXISTS (SELECT 1 from users WHERE email = $1) AS email_exists;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1;

-- name: SoftDeleteUser :exec
UPDATE users
SET
  is_deleted = TRUE,
  display_name = '[deleted]',
  deleted_at = NOW(),
  archived_username = username,
  archived_email    = email,
  username = CONCAT('deleted_user_', id),
  email    = CONCAT('deleted_', id, '@invalid.local'),
  profile_img_url = ''
WHERE id = $1 AND is_deleted = FALSE;
