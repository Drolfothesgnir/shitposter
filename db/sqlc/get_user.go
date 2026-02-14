package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const opGetUser = "get-user"

func (s *SQLStore) GetUser(ctx context.Context, userID int64) (User, error) {
	user, err := s.getUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, notFoundError(opGetUser, entUser, userID)
		}

		opErr := sqlError(
			opGetUser,
			opDetails{userID: userID, entity: entUser},
			err,
		)

		return User{}, opErr
	}

	// if user is soft-deleted return err
	if user.IsDeleted {
		opErr := newOpError(
			opGetUser,
			KindDeleted,
			entUser,
			fmt.Errorf("user with id %d is deleted", userID),
			withEntityID(userID),
		)

		return User{}, opErr
	}

	return user, nil
}
