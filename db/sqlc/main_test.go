package db

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
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

// will generate approximately 861 comment
func TestCreateDummyComments(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	GenerateDummyComments(t, CommentsGeneratorParams{
		AvailableUserIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		PostID:           4,
		MaxAttempts:      3,
		Count:            100,
	})

}

type CommentsGeneratorParams struct {
	AvailableUserIDs []int64
	PostID           int64
	MaxAttempts      int64
	Count            int64
}

// recursive function to create comment
func f(t *testing.T, parent_id *int64, prob float64, params CommentsGeneratorParams, wg *sync.WaitGroup) {
	n := rand.Int64N(int64(len(params.AvailableUserIDs)))

	body := "top comment"
	if parent_id != nil {
		body = fmt.Sprintf("reply to a comment %d", *parent_id)
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

	p_id, ok := getParentID(parent_id)

	comment, err := testStore.CreateComment(context.Background(), CreateCommentParams{
		PUserID:    params.AvailableUserIDs[n],
		PPostID:    params.PostID,
		PBody:      body,
		PParentID:  pgtype.Int8{Int64: p_id, Valid: ok},
		PUpvotes:   pgtype.Int8{Int64: int64(upvote_scalar * float64(rand.Int64N(1000))), Valid: true},
		PDownvotes: pgtype.Int8{Int64: int64(downvote_scalar * float64(rand.Int64N(1000))), Valid: true},
	})

	require.NoError(t, err)

	// attempt to create first level replies
	for range params.MaxAttempts {
		p := rand.Float64()
		if p < prob {
			wg.Add(1)
			go func() {
				defer wg.Done()
				f(t, &comment.ID, prob*0.5, params, wg)
			}()
		}
	}

}

func GenerateDummyComments(t *testing.T, params CommentsGeneratorParams) {

	var wg sync.WaitGroup

	for range params.Count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f(t, nil, 0.8, params, &wg)
		}()
	}

	wg.Wait()
}

func getParentID(parent_id *int64) (int64, bool) {
	if parent_id != nil {
		return *parent_id, true
	}

	return -1, false
}
