package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/stretchr/testify/require"
)

func createCredentialForUserWithSignCount(t *testing.T, userID int64, signCount int64) WebauthnCredential {
	t.Helper()

	arg := credentialParams(t, userID)
	arg.SignCount = signCount

	cred, err := testStore.createWebauthnCredentials(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, signCount, cred.SignCount)

	return cred
}

func TestRecordCredentialUseQuery_AdvancesCounterAndTouchesCredential(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	cred := createCredentialForUserWithSignCount(t, user.ID, 3)

	row, err := testStore.recordCredentialUse(ctx, recordCredentialUseParams{
		PCredID:    cred.ID,
		PSignCount: 8,
	})
	require.NoError(t, err)
	require.True(t, row.CredExists)
	require.EqualValues(t, 3, row.PrevCount)
	require.False(t, row.IsSuspicious)

	updated, err := testStore.getCredentialsByID(ctx, cred.ID)
	require.NoError(t, err)
	require.EqualValues(t, 8, updated.SignCount)
	require.True(t, updated.LastUsedAt.Valid)
	require.WithinDuration(t, time.Now(), updated.LastUsedAt.Time, 3*time.Second)
}

func TestRecordCredentialUseQuery_AllowsZeroToZero(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	cred := createCredentialForUserWithSignCount(t, user.ID, 0)

	row, err := testStore.recordCredentialUse(ctx, recordCredentialUseParams{
		PCredID:    cred.ID,
		PSignCount: 0,
	})
	require.NoError(t, err)
	require.True(t, row.CredExists)
	require.EqualValues(t, 0, row.PrevCount)
	require.False(t, row.IsSuspicious)

	updated, err := testStore.getCredentialsByID(ctx, cred.ID)
	require.NoError(t, err)
	require.EqualValues(t, 0, updated.SignCount)
	require.True(t, updated.LastUsedAt.Valid)
	require.WithinDuration(t, time.Now(), updated.LastUsedAt.Time, 3*time.Second)
}

func TestRecordCredentialUseQuery_FlagsSuspiciousAndPreservesState(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	cred := createCredentialForUserWithSignCount(t, user.ID, 5)

	row, err := testStore.recordCredentialUse(ctx, recordCredentialUseParams{
		PCredID:    cred.ID,
		PSignCount: 0,
	})
	require.NoError(t, err)
	require.True(t, row.CredExists)
	require.EqualValues(t, 5, row.PrevCount)
	require.True(t, row.IsSuspicious)

	updated, err := testStore.getCredentialsByID(ctx, cred.ID)
	require.NoError(t, err)
	require.EqualValues(t, 5, updated.SignCount)
	require.False(t, updated.LastUsedAt.Valid)
}

func TestRecordCredentialUseQuery_MissingCredential(t *testing.T) {
	ctx := context.Background()

	row, err := testStore.recordCredentialUse(ctx, recordCredentialUseParams{
		PCredID:    util.RandomByteArray(32),
		PSignCount: 7,
	})
	require.NoError(t, err)
	require.False(t, row.CredExists)
	require.EqualValues(t, -1, row.PrevCount)
	require.False(t, row.IsSuspicious)
}

func TestRecordCredentialUse_Success(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	cred := createCredentialForUserWithSignCount(t, user.ID, 0)

	err := testStore.RecordCredentialUse(ctx, RecordCredentialUseParams{
		ID:        cred.ID,
		SignCount: 0,
	})
	require.NoError(t, err)

	updated, err := testStore.getCredentialsByID(ctx, cred.ID)
	require.NoError(t, err)
	require.True(t, updated.LastUsedAt.Valid)
}

func TestRecordCredentialUse_NotFound(t *testing.T) {
	ctx := context.Background()
	missingID := util.RandomByteArray(32)

	err := testStore.RecordCredentialUse(ctx, RecordCredentialUseParams{
		ID:        missingID,
		SignCount: 9,
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opRecordCredentialUse, opErr.Op)
	require.Equal(t, KindNotFound, opErr.Kind)
	require.Equal(t, entWauthnCred, opErr.Entity)
	require.Equal(t, fmt.Sprintf("%x", missingID), opErr.EntityID)
}

func TestRecordCredentialUse_Suspicious(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	cred := createCredentialForUserWithSignCount(t, user.ID, 5)

	err := testStore.RecordCredentialUse(ctx, RecordCredentialUseParams{
		ID:        cred.ID,
		SignCount: 0,
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opRecordCredentialUse, opErr.Op)
	require.Equal(t, KindSecurity, opErr.Kind)
	require.Equal(t, entWauthnCred, opErr.Entity)
	require.Equal(t, fmt.Sprintf("%x", cred.ID), opErr.EntityID)
	require.Contains(t, opErr.Error(), "provided sign count 0 is not greater than current stored sign count 5")

	updated, getErr := testStore.getCredentialsByID(ctx, cred.ID)
	require.NoError(t, getErr)
	require.EqualValues(t, 5, updated.SignCount)
	require.False(t, updated.LastUsedAt.Valid)
}
