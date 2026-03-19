package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const opGetUserCredentials = "get-user-credentials"

// GetUserCredentials retrieves the Webauthn credentials for the user with provided ID.
// Returns [KindNotFound] if the credentials does not exist or [KindInternal] on database errors.
func (s *SQLStore) GetUserCredentials(ctx context.Context, userID int64) ([]WebauthnCredential, error) {
	creds, err := s.getUserCredentials(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			opErr := newOpError(
				opGetUserCredentials,
				KindNotFound,
				entWauthnCred,
				fmt.Errorf("webauthn credentials for the user with ID [%d] not found", userID),
				withUser(fmt.Sprint(userID)),
			)
			return []WebauthnCredential{}, opErr
		}

		return []WebauthnCredential{}, sqlError(
			opGetUserCredentials,
			opDetails{
				userID: fmt.Sprint(userID),
				entity: entWauthnCred,
			},
			err,
		)
	}

	return creds, nil
}
