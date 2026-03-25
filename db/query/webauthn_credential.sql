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

-- name: recordCredentialUse :one
SELECT 
  cred_exists::BOOLEAN AS cred_exists,
	prev_count::BIGINT AS prev_count,
	is_suspicious::BOOLEAN AS is_suspicious
FROM record_credential_use($1, $2);

-- name: deleteUserCredentials :exec
DELETE FROM webauthn_credentials
WHERE user_id = $1;

-- name: listUserCredentials :many
SELECT * FROM webauthn_credentials
WHERE user_id = $1;