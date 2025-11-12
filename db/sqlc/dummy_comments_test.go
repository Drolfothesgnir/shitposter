package db

import (
	"context"
	"errors"
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

	const (
		numUsers            = 10
		numRoots            = 100
		numWorkers          = 8
		prob                = 0.8
		decay               = 0.5
		maxRepliesPerParent = 3
	)

	users, err := testStore.TestUtilGetActiveUsers(context.Background(), numUsers)
	require.NoError(t, err)
	require.NotEmpty(t, users)

	nUsersActual := len(users)

	userIDs := make([]int64, nUsersActual)

	for i, user := range users {
		userIDs[i] = user.ID
	}

	post, err := testStore.GetNewestPosts(context.Background(), GetNewestPostsParams{
		Limit:  1,
		Offset: 0,
	})

	require.NoError(t, err)
	require.NotEmpty(t, post)

	postID := post[0].ID

	var wg sync.WaitGroup

	wg.Add(numRoots)

	a := maxRepliesPerParent * prob
	q := decay
	eps := 1e-9

	// Expected number of generated comments.
	// with these ↑ exact values it is expected to generate about 861 comment
	e, _, err := PartialTheta(a, q, eps)
	require.NoError(t, err)

	expComments := int(e) * numRoots

	queue := make([]Comment, numRoots, expComments)

	r := rand.New(rand.NewPCG(42, 1024))

	jobs := make(chan int)

	errQueue := make([]error, 0, expComments)

	errs := make(chan error)

	done := make(chan struct{})

	go func() {
		for err := range errs {
			if err != nil {
				errQueue = append(errQueue, err)
			}

		}
		done <- struct{}{}
	}()

	for w := range numWorkers {
		go func(w int) {
			r := rand.New(rand.NewPCG(uint64(w), 1024))
			for i := range jobs {
				j := r.Int64N(int64(len(userIDs)))
				c, err := createTestComment(r, postID, nil, userIDs[j])
				if err == nil {
					queue[i] = c
				}
				errs <- err
				wg.Done()
			}
		}(w)
	}

	go func() {
		for i := range numRoots {
			jobs <- i
		}
		close(jobs)
	}()

	wg.Wait()

	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		if cur.ID == 0 {
			continue
		}

		ids := slices.Clone(userIDs)
		attempts := min(maxRepliesPerParent, len(ids))
		for range attempts {
			p := r.Float64()
			if p < prob*math.Pow(decay, float64(cur.Depth)) {
				j := r.Int64N(int64(len(ids)))
				newComment, err := createTestComment(r, postID, &cur.ID, ids[j])
				if err == nil {
					queue = append(queue, newComment)
				}
				errs <- err
				ids = slices.Delete(ids, int(j), int(j)+1)
			}
		}
	}

	close(errs)
	<-done

	for _, err := range errQueue {
		require.NoError(t, err)
	}
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

	comment, err := testStore.CreateComment(context.Background(), CreateCommentParams{
		PUserID:    userID,
		PPostID:    postID,
		PBody:      body,
		PParentID:  pgtype.Int8{Int64: p_id, Valid: ok},
		PUpvotes:   pgtype.Int8{Int64: int64(upvote_scalar * float64(r.Int64N(1000))), Valid: true},
		PDownvotes: pgtype.Int8{Int64: int64(downvote_scalar * float64(r.Int64N(1000))), Valid: true},
	})

	return comment, err
}

func getParentID(parent_id *int64) (int64, bool) {
	if parent_id != nil {
		return *parent_id, true
	}

	return -1, false
}

// PartialTheta computes S(a,q) = sum_{n>=0} a^n q^{n(n-1)/2}
// with absolute error <= eps (roughly), for 0<q<1, a>=0.
// It returns the sum and the number of terms used.
func PartialTheta(a, q, eps float64) (sum float64, terms int, _ error) {
	if !(q > 0 && q < 1) {
		return 0, 0, errors.New("q must be in (0,1)")
	}
	if a < 0 || eps <= 0 {
		return 0, 0, errors.New("a >= 0 and eps > 0 required")
	}

	// Kahan compensated sum
	var c float64
	add := func(x float64) {
		y := x - c
		t := sum + y
		c = (t - sum) - y
		sum = t
	}

	// t_n is current term; start at n=0
	t := 1.0
	add(t)
	terms = 1

	// ratio for next step: r_n = a*q^{n}; start with n=0 => r=a
	r := a

	// Warm-up to reach r*q < 1 so tail bound becomes valid.
	// Solve a*q^N < 1 => N > log(1/a)/log(q) (note log(q)<0).
	if a >= 1 {
		N0 := int(math.Floor(math.Log(1/a)/math.Log(q))) + 1
		N0 = max(N0, 0)
		for i := 0; i < N0; i++ {
			t *= r // t_{n+1} = t_n * r_n
			add(t)
			terms++
			r *= q // r_{n+1} = a*q^{n+1}
		}
	}

	for {
		// advance one term
		t *= r
		add(t)
		terms++

		// Next-step upper bound on subsequent ratios is r*q (since ratios decrease).
		rBound := r * q
		if rBound < 1 {
			// Tail <= t_{next} * rBound / (1 - rBound)
			// Here t is t_{current}; next term would be t*rBound roughly,
			// but this bound is conservative and simple:
			tail := t * rBound / (1 - rBound)
			if tail <= eps {
				return sum, terms, nil
			}
		}

		// prepare next ratio
		r *= q
	}
}
