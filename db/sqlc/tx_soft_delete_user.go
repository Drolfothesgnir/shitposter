package db

import "context"

func (store *SQLStore) SoftDeleteUserTx(ctx context.Context, userID int64) error {
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		err = q.DeleteUserSessions(ctx, userID)
		if err != nil {
			return err
		}

		err = q.DeleteUserCredentials(ctx, userID)
		if err != nil {
			return err
		}

		_, err = q.softDeleteUser(ctx, userID)

		return err
	})
	return err
}
