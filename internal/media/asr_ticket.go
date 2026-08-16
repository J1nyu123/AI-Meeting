package media

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrTicketInvalid = errors.New("ASR ticket is invalid or expired")
	ErrASRInUse      = errors.New("an ASR session is already active")
)

type TicketStore interface {
	Issue(context.Context, uint64) (string, time.Time, error)
	Redeem(context.Context, string) (uint64, error)
}

type LeaseStore interface {
	Acquire(context.Context, uint64, string) error
	Renew(context.Context, uint64, string) error
	Release(context.Context, uint64, string)
}

type RedisASRStore struct {
	client    *redis.Client
	ticketTTL time.Duration
	leaseTTL  time.Duration
	now       func() time.Time
}

func NewRedisASRStore(client *redis.Client) *RedisASRStore {
	return &RedisASRStore{client: client, ticketTTL: time.Minute, leaseTTL: 30 * time.Second, now: time.Now}
}

func (s *RedisASRStore) Issue(ctx context.Context, userID uint64) (string, time.Time, error) {
	if s.client == nil {
		return "", time.Time{}, errors.New("redis is unavailable")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	ticket := base64.RawURLEncoding.EncodeToString(raw)
	if err := s.client.Set(ctx, "media:asr:ticket:"+ticket, userID, s.ticketTTL).Err(); err != nil {
		return "", time.Time{}, err
	}
	return ticket, s.now().Add(s.ticketTTL), nil
}

func (s *RedisASRStore) Redeem(ctx context.Context, ticket string) (uint64, error) {
	if s.client == nil || ticket == "" {
		return 0, ErrTicketInvalid
	}
	value, err := s.client.GetDel(ctx, "media:asr:ticket:"+ticket).Result()
	if err != nil {
		return 0, ErrTicketInvalid
	}
	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || userID == 0 {
		return 0, ErrTicketInvalid
	}
	return userID, nil
}

func (s *RedisASRStore) Acquire(ctx context.Context, userID uint64, owner string) error {
	if s.client == nil {
		return errors.New("redis is unavailable")
	}
	ok, err := s.client.SetNX(ctx, leaseKey(userID), owner, s.leaseTTL).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrASRInUse
	}
	return nil
}

var renewLeaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0`)
var releaseLeaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)

func (s *RedisASRStore) Renew(ctx context.Context, userID uint64, owner string) error {
	if s.client == nil {
		return errors.New("redis is unavailable")
	}
	result, err := renewLeaseScript.Run(ctx, s.client, []string{leaseKey(userID)}, owner, s.leaseTTL.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrASRInUse
	}
	return nil
}
func (s *RedisASRStore) Release(ctx context.Context, userID uint64, owner string) {
	if s.client != nil {
		_, _ = releaseLeaseScript.Run(ctx, s.client, []string{leaseKey(userID)}, owner).Result()
	}
}
func leaseKey(userID uint64) string { return fmt.Sprintf("media:asr:lease:%d", userID) }
