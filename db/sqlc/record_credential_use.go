package db

import (
	"context"
	"fmt"
)

type RecordCredentialUseParams struct {
	ID        []byte `json:"id"`
	SignCount int64  `json:"sign_count"`
}

const opRecordCredentialUse = "record-credential-use"

// RecordCredentialUse maintains sign-in related metadata of the webauthn credential, according
// to the policy:
//
// the credential's sign count and the last-used-at update will be considered valid only
//   - when the new provided sign count is greater than the current sign count for this credential OR
//   - when both the provided sign count and the current one are equal to zero, which is considered as a result
//     of the valid authentificator which cannot handle sign-in counter well.
//
// Errors returned:
//   - [KindNotFound] in case the credential with provided ID is not found.
//   - [KindSecurity] if the update is considred suspicious.
//   - [KindInternal] in case some other internal error.
//
// TODO: test generated and this file thoroughly
func (s *SQLStore) RecordCredentialUse(ctx context.Context, arg RecordCredentialUseParams) error {
	record_res, err := s.recordCredentialUse(ctx, recordCredentialUseParams{
		PCredID:    arg.ID,
		PSignCount: arg.SignCount,
	})

	if err != nil {
		return sqlError(
			opRecordCredentialUse,
			opDetails{
				entity:   entWauthnCred,
				entityID: fmt.Sprintf("%x", arg.ID),
				input:    fmt.Sprintf("%d", arg.SignCount),
			},
			err,
		)
	}

	if !record_res.CredExists {
		return notFoundError(
			opRecordCredentialUse,
			entWauthnCred,
			fmt.Sprintf("%x", arg.ID),
		)
	}

	if record_res.IsSuspicious {
		return newOpError(
			opRecordCredentialUse,
			KindSecurity,
			entWauthnCred,
			fmt.Errorf(
				"credential use rejected: provided sign count %d is not greater than current stored sign count %d; only a 0->0 non-advancing counter transition is allowed",
				arg.SignCount,
				record_res.PrevCount,
			),
			withEntityID(fmt.Sprintf("%x", arg.ID)),
		)
	}

	return nil
}
