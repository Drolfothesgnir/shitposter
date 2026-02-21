package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
)

const opVoteComment = "vote-comment"

type VoteCommentTxParams struct {
	UserID    int64
	CommentID int64
	Vote      int16 // Can be 1 or -1
}

// VoteCommentTx records an upvote (+1) or downvote (-1) on a comment and updates
// the comment's popularity counters within a transaction. Returns KindInvalid if
// the vote value is not 1 or -1, KindRelation if the comment or user does not exist,
// KindConflict if the same vote has already been cast, KindDeleted if the comment is
// soft-deleted, or KindInternal on database errors.
func (s *SQLStore) VoteCommentTx(ctx context.Context, arg VoteCommentTxParams) (Comment, error) {
	var result Comment
	err := s.execTx(ctx, func(q *Queries) error {
		// sanity check for the vote value
		if arg.Vote != -1 && arg.Vote != 1 {
			return newOpError(
				opVoteComment,
				KindInvalid,
				entCommentVote,
				fmt.Errorf("voting value is invalid: %d. must be either 1 or -1", arg.Vote),
				withField("vote"),
			)
		}

		row, err := q.upsertCommentVote(ctx, upsertCommentVoteParams{
			PUserID:    arg.UserID,
			PCommentID: arg.CommentID,
			PVote:      arg.Vote,
		})

		voteStr := strconv.Itoa(int(arg.Vote))

		// check if there are db violations to determine if either user id or comment id is invalid
		if err != nil {
			return sqlError(
				opVoteComment,
				opDetails{
					userID:    arg.UserID,
					commentID: arg.CommentID,
					entity:    entCommentVote,
					input:     voteStr,
				},
				err,
			)
		}

		oldVote := row.OriginalVote // -1, 0, 1 (0 = no previous voting)
		newVote := arg.Vote

		// repeated vote: don't change anything
		if oldVote == newVote {
			return newOpError(
				opVoteComment,
				KindConflict,
				entCommentVote,
				fmt.Errorf("repeated voting value: %d", arg.Vote),
				withField("vote"),
				withEntityID(row.ID),
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

		comment, err := q.updateCommentPopularity(ctx, updateCommentPopularityParams{
			ID:             arg.CommentID,
			UpvotesDelta:   upDelta,
			DownvotesDelta: downDelta,
		})

		// if 'not found' returned it means
		// the comment is deleted and cannot be voted
		if errors.Is(err, pgx.ErrNoRows) {
			return newOpError(
				opVoteComment,
				KindDeleted,
				entComment,
				fmt.Errorf("comment with id %d is deleted and cannot be voted", arg.CommentID),
				withEntityID(arg.CommentID),
			)
		}

		if err != nil {
			return sqlError(
				opVoteComment,
				opDetails{
					userID:    arg.UserID,
					commentID: arg.CommentID,
					entity:    entCommentVote,
					input:     voteStr,
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
