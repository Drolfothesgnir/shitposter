-- name: GetUserCredentials :many
SELECT * FROM webauthn_credentials
WHERE user_id = $1;