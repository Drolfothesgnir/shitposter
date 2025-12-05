-- name: getUserCredentials :many
SELECT * FROM webauthn_credentials
WHERE user_id = $1;

-- name: getCredentialsByID :one
SELECT * FROM webauthn_credentials
WHERE id = $1;

-- name: createWebauthnCredentials :one
INSERT INTO webauthn_credentials (
  id,
  user_id,                  
  public_key,               
  attestation_type,         
  transports,               
  user_present,             
  user_verified,            
  backup_eligible,          
  backup_state,             
  aaguid,                   
  sign_count,               
  clone_warning,            
  authenticator_attachment, 
  authenticator_data,       
  public_key_algorithm
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15
) RETURNING *;

-- name: UpdateCredentialSignCount :exec
UPDATE webauthn_credentials
SET
  sign_count = $2
WHERE id = $1;

-- name: deleteUserCredentials :exec
DELETE FROM webauthn_credentials
WHERE user_id = $1;

-- name: ListUserCredentials :many
SELECT * FROM webauthn_credentials
WHERE user_id = $1;