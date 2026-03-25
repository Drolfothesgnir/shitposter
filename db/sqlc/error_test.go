package db

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestSQLError_DefaultConflictPreservesEntityID(t *testing.T) {
	err := sqlError(
		"create-credential",
		opDetails{
			entity:   entWauthnCred,
			entityID: "cred-123",
			userID:   "42",
		},
		&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "webauthn_credentials_pkey",
		},
	)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, KindConflict, opErr.Kind)
	require.Equal(t, entWauthnCred, opErr.Entity)
	require.Equal(t, "cred-123", opErr.EntityID)
	require.Equal(t, "42", opErr.UserID)
}

func TestSQLError_UserConflictUsesGenericEntityID(t *testing.T) {
	err := sqlError(
		"update-user",
		opDetails{
			entity:   entUser,
			entityID: "7",
			input:    "taken@example.com",
		},
		&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "uniq_users_email_active",
		},
	)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, KindConflict, opErr.Kind)
	require.Equal(t, entUser, opErr.Entity)
	require.Equal(t, "7", opErr.EntityID)
	require.Equal(t, "email", opErr.FailingField)
}
