package db

import (
	"context"
	"fmt"
)

type UpdateCredentialSignCountParams struct {
	ID        []byte `json:"id"`
	SignCount int64  `json:"sign_count"`
}

const opUpdateCredentialSignCount = "update-credential-sign-count"

// TODO: add proper docs and expose this method
func (s *SQLStore) UpdateCredentialSignCount(ctx context.Context, arg UpdateCredentialSignCountParams) error {
	updated, err := s.updateCredentialSignCount(ctx, updateCredentialSignCountParams{
		ID:        arg.ID,
		SignCount: arg.SignCount,
	})

	if err != nil {
		return sqlError(
			opUpdateCredentialSignCount,
			opDetails{
				input: fmt.Sprintf("%x", arg.ID),
				// TODO: add generic entityID field, but keep userID field
				entity: entWauthnCred,
			},
			err,
		)
	}

	if !updated {
		return notFoundError(
			opUpdateCredentialSignCount,
			entWauthnCred,
			fmt.Sprintf("%x", arg.ID),
		)
	}

	return nil
}
