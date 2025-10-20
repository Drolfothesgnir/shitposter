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

-- name: DeleteUser :one
UPDATE users
  SET 
    is_deleted = true,
    display_name = '[deleted]',
    deleted_at = NOW()
WHERE id = $1
RETURNING *;
