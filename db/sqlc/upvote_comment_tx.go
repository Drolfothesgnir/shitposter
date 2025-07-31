package db

import "context"

type UpvoteCommentTxParams struct {
	UserID    int64 `json:"user_id"`
	CommentID int64 `json:"comment_id"`
}

type UpvoteCommentTxResult struct {
	Comment     Comment     `json:"comment"`
	CommentVote CommentVote `json:"comment_vote"`
}

func (store *SQLStore) UpvoteCommentTx(ctx context.Context, arg UpvoteCommentTxParams) (UpvoteCommentTxResult, error) {
	var result UpvoteCommentTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.CommentVote, err = q.CreateCommentVote(ctx, CreateCommentVoteParams{
			UserID:    arg.UserID,
			CommentID: arg.CommentID,
			Vote:      1,
		})

		if err != nil {
			return err
		}

		result.Comment, err = q.UpvoteComment(ctx, arg.CommentID)

		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}
