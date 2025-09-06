package db

import "context"

type CreateCredentialsTxParams struct {
	ID         []byte `json:"id"`
	PublicKey  []byte `json:"public_key"`
	SignCount  int64  `json:"sign_count"`
	Transports []byte `json:"transports"`
}

type CreateUserWithCredentialsTxParams struct {
	User CreateUserParams          `json:"user"`
	Cred CreateCredentialsTxParams `json:"cred"`
}

type CreateUserWithCredentialsTxResult struct {
	User User `json:"user"`
}

// Function to create both "users" row and "webauthn_credentials" row in one transaction.
func (store *SQLStore) CreateUserWithCredentialsTx(ctx context.Context, arg CreateUserWithCredentialsTxParams) (CreateUserWithCredentialsTxResult, error) {
	var result CreateUserWithCredentialsTxResult
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		result.User, err = q.CreateUser(ctx, arg.User)
		if err != nil {
			return err
		}

		params := CreateWebauthnCredentialsParams{
			ID:         arg.Cred.ID,
			UserID:     result.User.ID,
			PublicKey:  arg.Cred.PublicKey,
			SignCount:  arg.Cred.SignCount,
			Transports: arg.Cred.Transports,
		}

		_, err = q.CreateWebauthnCredentials(ctx, params)
		return err
	})

	return result, err
}
