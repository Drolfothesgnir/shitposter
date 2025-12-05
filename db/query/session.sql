-- name: createSession :one
INSERT INTO sessions (
  id,
  user_id,
  refresh_token,
  user_agent,
  client_ip,
  is_blocked,
  expires_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: getSession :one
SELECT * FROM sessions
WHERE id = $1 LIMIT 1;

-- name: deleteUserSessions :exec
DELETE FROM sessions
WHERE user_id = $1;

-- name: listSessionsByUser :many
SELECT * FROM sessions
WHERE user_id = $1;