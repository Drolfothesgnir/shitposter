package db

import "context"

const (
	opUsernameExists = "username-exists"
	opEmailExists    = "email-exists"
)

// UsernameExists checks if user with provided username exists and returns boolean and error.
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

// EmailExists checks if user with provided email exists and returns boolean and error.
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
