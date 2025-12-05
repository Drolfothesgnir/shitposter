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

-- name: getUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: UsernameExists :one 
SELECT EXISTS (SELECT 1 from users WHERE username = $1) AS username_exists;

-- name: EmailExists :one 
SELECT EXISTS (SELECT 1 from users WHERE email = $1) AS email_exists;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1;

-- name: softDeleteUser :one
SELECT 
  id::BIGINT AS id,
  username::TEXT AS username,
  display_name::TEXT AS display_name,
  email::TEXT AS email,
  profile_img_url::optional_string AS profile_img_url ,
  is_deleted::BOOLEAN AS is_deleted,
  deleted_at::TIMESTAMPTZ AS deleted_at,
  last_modified_at::TIMESTAMPTZ AS last_modified_at,
  success::BOOLEAN AS success
FROM soft_delete_user(
  p_user_id := $1
);

-- name: updateUser :one
SELECT 
  id::BIGINT AS id,
	username::TEXT AS username,
	email::TEXT AS email,
	profile_img_url::optional_string AS profile_img_url,
	is_deleted::BOOLEAN AS is_deleted,
	last_modified_at::TIMESTAMPTZ AS last_modified_at,
	updated::BOOLEAN AS updated
FROM update_user(
  p_user_id := $1,
  p_username := sqlc.narg(username),
  p_email := sqlc.narg(email),
  p_profile_img_url := sqlc.narg(profile_img_url)
);

-- name: TestUtilGetActiveUsers :many
SELECT * FROM users
WHERE is_deleted = FALSE
LIMIT $1;

-- name: getUserByEmail :one
SELECT * FROM users
WHERE email = $1;