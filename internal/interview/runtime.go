package interview

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RuntimeStore struct {
	redis *redis.Client
	ttl   time.Duration
}

type runtimeStateReaderWriter interface {
	Get(context.Context, string) (RuntimeState, error)
	Set(context.Context, RuntimeState) error
}

func NewRuntimeStore(client *redis.Client) *RuntimeStore {
	return &RuntimeStore{redis: client, ttl: 24 * time.Hour}
}
func (s *RuntimeStore) key(id string) string { return "interview:runtime:" + id }
func (s *RuntimeStore) Get(ctx context.Context, id string) (RuntimeState, error) {
	if s.redis == nil {
		return RuntimeState{}, redis.Nil
	}
	raw, err := s.redis.Get(ctx, s.key(id)).Bytes()
	if err != nil {
		return RuntimeState{}, err
	}
	var state RuntimeState
	err = json.Unmarshal(raw, &state)
	if err == nil {
		s.redis.Expire(ctx, s.key(id), s.ttl)
	}
	return state, err
}
func (s *RuntimeStore) Set(ctx context.Context, state RuntimeState) error {
	if s.redis == nil {
		return errors.New("redis disabled")
	}
	raw, _ := json.Marshal(state)
	return s.redis.Set(ctx, s.key(state.SessionID), raw, s.ttl).Err()
}
func (s *RuntimeStore) Delete(ctx context.Context, id string) error {
	if s.redis == nil {
		return errors.New("redis disabled")
	}
	return s.redis.Del(ctx, s.key(id)).Err()
}
func (s *RuntimeStore) Acquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if s.redis == nil {
		return false, errors.New("redis disabled")
	}
	return s.redis.SetNX(ctx, "interview:lock:"+key, owner, ttl).Result()
}
func (s *RuntimeStore) Release(ctx context.Context, key, owner string) {
	if s.redis == nil {
		return
	}
	s.redis.Eval(ctx, `if redis.call("get",KEYS[1])==ARGV[1] then return redis.call("del",KEYS[1]) else return 0 end`, []string{"interview:lock:" + key}, owner)
}

func runtimeStateFromSession(session Session) RuntimeState {
	return RuntimeState{SessionID: session.ID, Status: session.Status, CurrentQuestionNumber: session.CurrentQuestionNumber, CurrentMainIndex: session.CurrentMainIndex, FollowUpCount: session.FollowUpCount, TotalScore: session.TotalScore, TurnSequence: session.TurnSequence, Version: session.Version, UpdatedAt: session.UpdatedAt}
}

func resolveRuntimeState(cached RuntimeState, cacheErr error, session Session) (RuntimeState, bool) {
	if cacheErr == nil && cached.SessionID == session.ID && cached.Version == session.Version {
		return cached, false
	}
	return runtimeStateFromSession(session), true
}

func Rehydrate(ctx context.Context, store runtimeStateReaderWriter, session Session) (RuntimeState, bool, error) {
	cached, cacheErr := store.Get(ctx, session.ID)
	state, rebuilt := resolveRuntimeState(cached, cacheErr, session)
	if !rebuilt {
		return state, false, nil
	}
	_ = store.Set(ctx, state)
	return state, true, nil
}
