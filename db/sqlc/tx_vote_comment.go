package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type VoteCommentTxParams struct {
	UserID    int64
	CommentID int64
	Vote      int16
}

func (s *SQLStore) VoteCommentTx(ctx context.Context, arg VoteCommentTxParams) (Comment, error) {
	var result Comment
	err := s.execTx(ctx, func(q *Queries) error {
		row, err := q.UpsertCommentVote(ctx, UpsertCommentVoteParams{
			PUserID:    arg.UserID,
			PCommentID: arg.CommentID,
			PVote:      arg.Vote,
		})

		// check if there are db violations to determine if either user id or post id is invalid
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case "comment_votes_comment_id_fkey":
				return ErrInvalidCommentID // keep shitty errors for now. i'll replace them with the Proper One
			case "comment_votes_user_id_fkey":
				return ErrInvalidUserID // and it's gonna be awesome!
			default:
				return err
			}
		}

		if err != nil {
			return err
		}

		var upDelta, downDelta int16

		// if the vote is repeated, the same as the old one
		// abort
		if !row.Delta {
			return ErrDuplicateVote
		}

		switch arg.Vote {
		case -1:
			// negative vote -> always add 1 to the downvotes
			downDelta = 1
			// if the user changed his vote from positive to negative
			// also remove 1 from the upvotes
			if !row.InsertedOk {
				upDelta = -1
			}
		case 1:
			// positive vote -> always add 1 to the upvotes
			upDelta = 1
			// if the user changed his vote from negative to positive
			// also remove 1 from the downvotes
			if !row.InsertedOk {
				downDelta = -1
			}
			// if the provided vote is not 1 or -1 abort
		default:
			return ErrInvalidVoteValue
		}

		comment, err := q.UpdateCommentPopularity(ctx, UpdateCommentPopularityParams{
			ID:             arg.CommentID,
			UpvotesDelta:   upDelta,
			DownvotesDelta: downDelta,
		})

		// if 'not found' returned it means
		// the comment is deleted and cannot be voted
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEntityDeleted // another bad one
		}

		if err != nil {
			return err
		}

		result = comment

		return nil
	})

	if err != nil {
		return Comment{}, err
	}

	return result, nil
}
