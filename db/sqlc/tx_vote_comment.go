package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
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
			return baseError(
				"vote-comment",
				"comment_vote",
				KindInvalid,
				fmt.Errorf("voting value is invalid: %d. must be either 1 or -1", arg.Vote),
			)
		}

		row, err := q.UpsertCommentVote(ctx, UpsertCommentVoteParams{
			PUserID:    arg.UserID,
			PCommentID: arg.CommentID,
			PVote:      arg.Vote,
		})

		// check if there are db violations to determine if either user id or comment id is invalid
		if err != nil {
			return sqlError(
				"vote-comment",
				opDetails{
					userID:    arg.UserID,
					commentID: arg.CommentID,
					entity:    "comment_vote",
					input:     strconv.Itoa(int(arg.Vote)),
				},
				err,
			)
		}

		oldVote := row.OriginalVote // -1, 0, 1 (0 = no previous voting)
		newVote := arg.Vote

		// repeated vote: don't change anything
		if oldVote == newVote {
			return baseError(
				"vote-comment",
				"comment_vote",
				KindConflict,
				fmt.Errorf("repeated voting value: %d", arg.Vote),
			)
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
			return withEntityID(
				baseError(
					"vote-comment",
					"comment",
					KindDeleted,
					fmt.Errorf("comment with id %d is deleted and cannot be voted", arg.CommentID),
				),
				arg.CommentID,
			)
		}

		if err != nil {
			return sqlError(
				"vote-comment",
				opDetails{
					userID:    arg.UserID,
					commentID: arg.CommentID,
					entity:    "comment",
				},
				err,
			)
		}

		result = comment

		return nil
	})

	if err != nil {
		return Comment{}, err
	}

	return result, nil
}
