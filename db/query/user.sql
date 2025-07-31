-- name: CreateUser :one
INSERT INTO users (
  username, 
  hashed_password,
  profile_img_url,
  email
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;