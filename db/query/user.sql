-- name: createUser :one
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

-- name: UpdateUser :one
UPDATE users
SET
  username = COALESCE(sqlc.narg(username), username),
  display_name = COALESCE(sqlc.narg(username), display_name),
  archived_username = COALESCE(sqlc.narg(username), archived_username),
  email = COALESCE(sqlc.narg(email), email),
  archived_email = COALESCE(sqlc.narg(email), archived_email),
  profile_img_url = COALESCE(sqlc.narg(profile_img_url), profile_img_url),
  last_modified_at = NOW()
WHERE id = $1 AND is_deleted = FALSE
RETURNING *;

-- name: TestUtilGetActiveUsers :many
SELECT * FROM users
WHERE is_deleted = FALSE
LIMIT $1;

-- name: getUserByEmail :one
SELECT * FROM users
WHERE email = $1;