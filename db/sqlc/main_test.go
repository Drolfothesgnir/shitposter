package db

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var testStore Store

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../../")
	if err != nil {
		log.Fatal("Cannot read the config: ", err)
	}

	connPool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal("Cannot connect to the database: ", err)
	}

	testStore = NewStore(connPool)

	os.Exit(m.Run())
}

func TestCreateDummyComments(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	nUsers := int32(10)

	users, err := testStore.TestUtilGetActiveUsers(context.Background(), nUsers)
	require.NoError(t, err)

	nUsersActual := len(users)

	userIds := make([]int64, nUsersActual)

	for i, user := range users {
		userIds[i] = user.ID
	}

	post, err := testStore.GetNewestPosts(context.Background(), GetNewestPostsParams{
		Limit:  1,
		Offset: 0,
	})

	postID := post[0].ID

	require.NoError(t, err)

	nRoots := 100

	var wg sync.WaitGroup

	wg.Add(nRoots)

	queue := make([]Comment, nRoots)

	for i := range queue {
		go func(i int) {
			defer wg.Done()
			j := rand.Int64N(int64(len(userIds)))
			queue[i] = createTestComment(t, postID, nil, userIds[j])
		}(i)
	}

	wg.Wait()

	prob := 0.8
	maxAttempts := 3

	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		ids := slices.Clone(userIds)
		for range maxAttempts {
			p := rand.Float64()
			if p < prob*math.Pow(0.5, float64(cur.Depth)) {
				j := rand.Int64N(int64(len(ids)))
				newComment := createTestComment(t, postID, &cur.ID, ids[j])
				ids = slices.Delete(ids, int(j), int(j)+1)
				queue = append(queue, newComment)
			}
		}
	}
}

func createTestComment(t *testing.T, postID int64, parentID *int64, userID int64) Comment {
	body := "top comment"
	if parentID != nil {
		body = fmt.Sprintf("reply to a comment %d", *parentID)
	}

	upvote_scalar := 1.0
	downvote_scalar := 1.0

	rand_float := rand.Float64()

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

	comment, err := testStore.CreateComment(context.Background(), CreateCommentParams{
		PUserID:    userID,
		PPostID:    postID,
		PBody:      body,
		PParentID:  pgtype.Int8{Int64: p_id, Valid: ok},
		PUpvotes:   pgtype.Int8{Int64: int64(upvote_scalar * float64(rand.Int64N(1000))), Valid: true},
		PDownvotes: pgtype.Int8{Int64: int64(downvote_scalar * float64(rand.Int64N(1000))), Valid: true},
	})

	require.NoError(t, err)

	return comment
}

func getParentID(parent_id *int64) (int64, bool) {
	if parent_id != nil {
		return *parent_id, true
	}

	return -1, false
}
