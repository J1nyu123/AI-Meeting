package media

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestTicketFailsClosedWithoutRedis(t *testing.T) {
	store := NewRedisASRStore(nil)
	_, _, err := store.Issue(context.Background(), 7)
	require.Error(t, err)
	_, err = store.Redeem(context.Background(), "ticket")
	require.ErrorIs(t, err, ErrTicketInvalid)
}

func TestTicketCanOnlyBeRedeemedOnce(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisASRStore(client)
	store.ticketTTL = time.Minute
	ticket, expiresAt, err := store.Issue(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, expiresAt.After(time.Now()))
	userID, err := store.Redeem(context.Background(), ticket)
	require.NoError(t, err)
	require.Equal(t, uint64(42), userID)
	_, err = store.Redeem(context.Background(), ticket)
	require.ErrorIs(t, err, ErrTicketInvalid)
}

func TestASRLeaseEnforcesSingleOwnerAndOwnerSafeRelease(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisASRStore(client)
	require.NoError(t, store.Acquire(context.Background(), 42, "owner-a"))
	require.ErrorIs(t, store.Acquire(context.Background(), 42, "owner-b"), ErrASRInUse)
	store.Release(context.Background(), 42, "owner-b")
	require.ErrorIs(t, store.Acquire(context.Background(), 42, "owner-b"), ErrASRInUse)
	require.NoError(t, store.Renew(context.Background(), 42, "owner-a"))
	store.Release(context.Background(), 42, "owner-a")
	require.NoError(t, store.Acquire(context.Background(), 42, "owner-b"))
}
