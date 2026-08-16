package media

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeTTSProvider struct {
	mu       sync.Mutex
	creates  int
	status   string
	audioURL string
}

func (p *fakeTTSProvider) Create(context.Context, TTSRequest) (ProviderTTSTask, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.creates++
	return ProviderTTSTask{TaskID: "provider-1", Status: "1", Code: 0}, nil
}
func (p *fakeTTSProvider) Query(context.Context, string) (ProviderTTSTask, error) {
	return ProviderTTSTask{TaskID: "provider-1", Status: p.status, Code: 0, AudioURL: p.audioURL}, nil
}
func testTTSService(t *testing.T, provider TTSProvider) *TTSService {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&TTSTask{}))
	return NewTTSService(db, provider, true, time.Second)
}
func TestTTSCreateReplaysIdempotentTask(t *testing.T) {
	provider := &fakeTTSProvider{}
	service := testTTSService(t, provider)
	first, err := service.Create(context.Background(), 7, "stable-key", TTSRequest{Text: "你好"})
	require.NoError(t, err)
	second, err := service.Create(context.Background(), 7, "stable-key", TTSRequest{Text: "不同文本也不应重复计费"})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, 1, provider.creates)
}
func TestTTSTaskOwnershipAndAudioReadiness(t *testing.T) {
	provider := &fakeTTSProvider{status: "5", audioURL: "https://audio.example.com/a.mp3"}
	service := testTTSService(t, provider)
	task, err := service.Create(context.Background(), 7, "stable-key", TTSRequest{Text: "你好"})
	require.NoError(t, err)
	_, err = service.Get(context.Background(), 8, task.ID, true)
	require.Error(t, err)
	task, err = service.Get(context.Background(), 7, task.ID, true)
	require.NoError(t, err)
	require.True(t, task.Success())
	raw, _, err := service.AudioURL(context.Background(), 7, task.ID)
	require.NoError(t, err)
	require.Equal(t, provider.audioURL, raw)
}
func TestTTSValidation(t *testing.T) {
	req := defaults(TTSRequest{Text: ""})
	require.Error(t, validateTTSRequest(req))
	req = defaults(TTSRequest{Text: "ok"})
	bad := 101
	req.Speed = &bad
	require.Error(t, validateTTSRequest(req))
}

func TestTTSTaskUsesMigrationSIDColumn(t *testing.T) {
	service := testTTSService(t, &fakeTTSProvider{})
	require.True(t, service.db.Migrator().HasColumn(&TTSTask{}, "sid"))
	require.False(t, service.db.Migrator().HasColumn(&TTSTask{}, "s_id"))
}
