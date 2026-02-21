package db

import "context"

const (
	opUsernameExists = "username-exists"
	opEmailExists    = "email-exists"
)

// UsernameExists reports whether an active user with the given username exists.
// Returns KindInternal on database errors.
func (s *SQLStore) UsernameExists(ctx context.Context, username string) (bool, error) {
	exists, err := s.usernameExists(ctx, username)
	if err != nil {
		return false, sqlError(
			opUsernameExists,
			opDetails{input: username, entity: entUser},
			err,
		)
	}

	return exists, nil
}

// EmailExists reports whether an active user with the given email exists.
// Returns KindInternal on database errors.
func (s *SQLStore) EmailExists(ctx context.Context, email string) (bool, error) {
	exists, err := s.emailExists(ctx, email)

	if err != nil {
		return false, sqlError(
			opEmailExists,
			opDetails{input: email, entity: entUser},
			err,
		)
	}

	return exists, nil
}
