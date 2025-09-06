-- name: GetUserCredentials :many
SELECT * FROM webauthn_credentials
WHERE user_id = $1;

-- name: CreateWebauthnCredentials :one
INSERT INTO webauthn_credentials (
  id,
  user_id,
  public_key,
  sign_count,
  transports
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;