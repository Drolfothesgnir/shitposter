package db

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateDummyComments(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// with this ↓ exact values it's expected to generate about 861 comments
	const (
		numUsers            = 10  // number of available user IDs
		numRoots            = 100 // number of top level (depth-0) comments
		numWorkers          = 8   // max number of dedicated goroutines
		prob                = 0.8 // base probability of creation of a reply
		decay               = 0.5 // by how much probability of creation will be reduced based on depth -> prob*decay^depth
		maxRepliesPerParent = 3   // number of same depth replies to any comment
	)

	// getting available user ids
	users, err := testStore.TestUtilGetActiveUsers(context.Background(), numUsers)
	require.NoError(t, err)
	require.NotEmpty(t, users)

	nUsersActual := len(users)

	userIDs := make([]int64, nUsersActual)

	for i, user := range users {
		userIDs[i] = user.ID
	}

	// getting active post to add comments to
	post, err := testStore.GetNewestPosts(context.Background(), GetNewestPostsParams{
		Limit:  1,
		Offset: 0,
	})

	require.NoError(t, err)
	require.NotEmpty(t, post)

	postID := post[0].ID

	var wg sync.WaitGroup

	// global RNG for reproducibility
	R := rand.New(rand.NewPCG(42, 1024))

	// task queue through which all work will happen.
	// it's critical to have buffered channel to avoid deadlocks.
	tasks := make(chan Comment, 1024)

	for w := range numWorkers {
		go func(i int) {
			// each worker will have its own RNG
			r := rand.New(rand.NewPCG(uint64(i), 1024))
			for task := range tasks {
				// the deeper the comment is the lower is the probability of reply creation
				threshold := prob * math.Pow(decay, float64(task.Depth))
				// guarding against out of bounds index for user id
				attempts := min(maxRepliesPerParent, len(userIDs))
				// cloning user ids to be able to remove already used id.
				// db doesn't allow multiple replies from the same user
				ids := slices.Clone(userIDs)
				for range attempts {
					p := r.Float64()
					if p > threshold {
						continue
					}

					j := r.Int64N(int64(len(ids)))
					userID := ids[j]
					cmnt, err := createTestComment(R, postID, &task.ID, userID)
					if err != nil {
						t.Log(err)
						continue
					}

					// removing used user id
					ids = slices.Delete(ids, int(j), int(j)+1)
					wg.Add(1)
					// sending back to job queue
					tasks <- cmnt
				}
				wg.Done()
			}
		}(w)
	}

	for range numRoots {
		i := R.Int64N(int64(len(userIDs)))
		root, err := createTestComment(R, postID, nil, userIDs[i])
		if err != nil {
			t.Log(err)
			continue
		}

		wg.Add(1)
		tasks <- root
	}

	wg.Wait()
}

func createTestComment(r *rand.Rand, postID int64, parentID *int64, userID int64) (Comment, error) {
	body := "top comment"
	if parentID != nil {
		body = fmt.Sprintf("reply to a comment %d", *parentID)
	}

	upvote_scalar := 1.0
	downvote_scalar := 1.0

	rand_float := r.Float64()

	// 33.33% chance of comment to be "popular"
	if rand_float < 1.0/3 {
		upvote_scalar = 2.0
		downvote_scalar = 0.5
	} else if rand_float > 1.0/3 && rand_float < 2.0/3 {
		// 33.33% chance of comment to be "unpopular"
		upvote_scalar = 0.5
		downvote_scalar = 2.0
	}

	p_id, ok := getParentID(parentID)

	comment, err := testStore.InsertCommentTx(context.Background(), InsertCommentTxParams{
		UserID:    userID,
		PostID:    postID,
		Body:      body,
		ParentID:  pgtype.Int8{Int64: p_id, Valid: ok},
		Upvotes:   int64(upvote_scalar * float64(r.Int64N(1000))),
		Downvotes: int64(downvote_scalar * float64(r.Int64N(1000))),
	})

	return comment, err
}

func getParentID(parent_id *int64) (int64, bool) {
	if parent_id != nil {
		return *parent_id, true
	}

	return -1, false
}
