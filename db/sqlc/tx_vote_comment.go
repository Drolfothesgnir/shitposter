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
	Vote      int16 // Can be 1 or -1
}

func (s *SQLStore) VoteCommentTx(ctx context.Context, arg VoteCommentTxParams) (Comment, error) {
	var result Comment
	err := s.execTx(ctx, func(q *Queries) error {
		// sanity check for the vote value
		if arg.Vote != -1 && arg.Vote != 1 {
			return ErrInvalidVoteValue
		}

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

		oldVote := row.OriginalVote // -1, 0, 1 (0 = no previous voting)
		newVote := arg.Vote

		// repeated vote: don't change anything
		if oldVote == newVote {
			return ErrDuplicateVote
		}

		var upDelta, downDelta int16

		// applying new vote effect
		switch newVote {
		case 1:
			upDelta++
		case -1:
			downDelta++
		}

		// removing old voting effect
		switch oldVote {
		case 1:
			upDelta--
		case -1:
			downDelta--
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
