package db

import "context"

type DownvoteCommentTxParams struct {
	UserID    int64 `json:"user_id"`
	CommentID int64 `json:"comment_id"`
}

type DownvoteCommentTxResult struct {
	Comment     Comment     `json:"comment"`
	CommentVote CommentVote `json:"comment_vote"`
}

func (store *SQLStore) DownvoteCommentTx(ctx context.Context, arg DownvoteCommentTxParams) (DownvoteCommentTxResult, error) {
	var result DownvoteCommentTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.CommentVote, err = q.CreateCommentVote(ctx, CreateCommentVoteParams{
			UserID:    arg.UserID,
			CommentID: arg.CommentID,
			Vote:      -1,
		})

		if err != nil {
			return err
		}

		result.Comment, err = q.DownvoteComment(ctx, arg.CommentID)

		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}
