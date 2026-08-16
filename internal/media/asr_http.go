package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"ai-meeting-go/internal/auth"
	"ai-meeting-go/internal/platform/httpx"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ASRHandler struct {
	Tickets     TicketStore
	Leases      LeaseStore
	Provider    ASRProvider
	Enabled     bool
	MaxDuration time.Duration
	IdleTimeout time.Duration
}

func (h *ASRHandler) IssueTicket(c *gin.Context) {
	if !h.Enabled {
		httpx.Fail(c, httpx.Conflict("MEDIA_ASR_DISABLED", "remote speech recognition is disabled"))
		return
	}
	ticket, expiresAt, err := h.Tickets.Issue(c, auth.UserID(c))
	if err != nil {
		httpx.Fail(c, &httpx.AppError{Status: http.StatusServiceUnavailable, Code: "ASR_TICKET_UNAVAILABLE", Message: "speech recognition is temporarily unavailable", Cause: err})
		return
	}
	httpx.Created(c, gin.H{"ticket": ticket, "expiresAt": expiresAt.UTC().Format(time.RFC3339Nano), "websocketUrl": "/api/v1/media/asr/ws?ticket=" + ticket})
}

func (h *ASRHandler) WebSocket(c *gin.Context) {
	userID, err := h.Tickets.Redeem(c, c.Query("ticket"))
	if err != nil {
		httpx.Fail(c, httpx.Unauthorized("invalid or expired ASR ticket"))
		return
	}
	owner := uuid.NewString()
	if err := h.Leases.Acquire(c, userID, owner); err != nil {
		status, code := http.StatusServiceUnavailable, "ASR_LEASE_UNAVAILABLE"
		if errors.Is(err, ErrASRInUse) {
			status, code = http.StatusConflict, "ASR_ALREADY_ACTIVE"
		}
		httpx.Fail(c, &httpx.AppError{Status: status, Code: code, Message: "speech recognition session is unavailable", Cause: err})
		return
	}
	defer h.Leases.Release(context.Background(), userID, owner)
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: []string{"localhost:*", "127.0.0.1:*"}})
	if err != nil {
		return
	}
	conn.SetReadLimit(32 << 10)
	ctx, cancel := context.WithTimeout(c.Request.Context(), durationOr(h.MaxDuration, 15*time.Minute))
	defer cancel()
	s := &asrSession{conn: conn, provider: h.Provider, userID: userID, owner: owner, leases: h.Leases, audio: make(chan []byte, 64), out: make(chan ASREvent, 64), done: make(chan error, 1), sessionID: uuid.NewString(), idleTimeout: durationOr(h.IdleTimeout, 45*time.Second)}
	s.run(ctx)
}

type asrSession struct {
	conn        *websocket.Conn
	provider    ASRProvider
	userID      uint64
	owner       string
	leases      LeaseStore
	audio       chan []byte
	out         chan ASREvent
	done        chan error
	sessionID   string
	idleTimeout time.Duration
	mu          sync.Mutex
	started     bool
	audioClosed bool
}

func (s *asrSession) run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer s.conn.CloseNow()
	writerDone := make(chan struct{})
	go s.writePump(ctx, writerDone)
	go s.renewLease(ctx, cancel)
	s.emit(ASREvent{Type: "connected", Code: "OK", SessionID: s.sessionID, Timestamp: time.Now().UnixMilli()})
	for {
		readCtx, readCancel := context.WithTimeout(ctx, s.idleTimeout)
		kind, payload, err := s.conn.Read(readCtx)
		readCancel()
		if err != nil {
			if errors.Is(readCtx.Err(), context.DeadlineExceeded) {
				s.emitError("ASR_IDLE_TIMEOUT", "speech recognition session timed out", true)
			}
			s.stopAudio()
			cancel()
			<-writerDone
			return
		}
		switch kind {
		case websocket.MessageText:
			var command struct {
				Type string `json:"type"`
			}
			if jsonError(payload, &command) != nil {
				s.emitError("INVALID_CONTROL", "invalid control message", false)
				continue
			}
			switch command.Type {
			case "ping":
				s.emit(ASREvent{Type: "pong", Code: "OK", SessionID: s.sessionID, Timestamp: time.Now().UnixMilli()})
			case "get_status":
				s.emit(ASREvent{Type: "status", Code: "OK", SessionID: s.sessionID, Message: s.status(), Timestamp: time.Now().UnixMilli()})
			case "start_transcription":
				s.startProvider(ctx)
			case "stop_transcription":
				if !s.hasStarted() {
					s.emit(ASREvent{Type: "transcription_stopped", Code: "OK", SessionID: s.sessionID, Timestamp: time.Now().UnixMilli()})
					time.Sleep(20 * time.Millisecond)
					cancel()
					<-writerDone
					return
				}
				s.stopAudio()
				select {
				case err := <-s.done:
					if err != nil {
						s.emitError("ASR_UPSTREAM_FAILED", "speech recognition failed", true)
					}
				case <-time.After(5 * time.Second):
					s.emitError("ASR_FINAL_TIMEOUT", "timed out waiting for final transcription", true)
				}
				s.emit(ASREvent{Type: "transcription_stopped", Code: "OK", SessionID: s.sessionID, Timestamp: time.Now().UnixMilli()})
				time.Sleep(20 * time.Millisecond)
				cancel()
				<-writerDone
				return
			default:
				s.emitError("UNKNOWN_CONTROL", "unknown control message", false)
			}
		case websocket.MessageBinary:
			if !s.isStarted() {
				s.emitError("ASR_NOT_STARTED", "start_transcription is required before audio", false)
				continue
			}
			chunk := append([]byte(nil), payload...)
			select {
			case s.audio <- chunk:
			case <-time.After(2 * time.Second):
				s.emitError("ASR_OVERLOADED", "audio buffer is full", true)
				_ = s.conn.Close(websocket.StatusTryAgainLater, "audio buffer full")
				cancel()
				<-writerDone
				return
			case <-ctx.Done():
				return
			}
		}
	}
}

func (s *asrSession) startProvider(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		s.emitError("ASR_ALREADY_STARTED", "transcription already started", false)
		return
	}
	s.started = true
	s.mu.Unlock()
	s.emit(ASREvent{Type: "transcription_started", Code: "OK", SessionID: s.sessionID, Timestamp: time.Now().UnixMilli()})
	go func() {
		err := s.provider.Run(ctx, s.sessionID, s.audio, s.out)
		if err != nil {
			s.emitError("ASR_UPSTREAM_FAILED", "speech recognition provider failed", true)
		}
		s.done <- err
	}()
}
func (s *asrSession) stopAudio() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started && !s.audioClosed {
		close(s.audio)
		s.audioClosed = true
	}
}
func (s *asrSession) isStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started && !s.audioClosed
}
func (s *asrSession) hasStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}
func (s *asrSession) status() string {
	if s.isStarted() {
		return "running"
	}
	return "connected"
}
func (s *asrSession) emit(e ASREvent) {
	select {
	case s.out <- e:
	default:
	}
}
func (s *asrSession) emitError(code, message string, retryable bool) {
	s.emit(ASREvent{Type: "error", Code: code, SessionID: s.sessionID, Message: message, Retryable: retryable, Timestamp: time.Now().UnixMilli()})
}
func (s *asrSession) writePump(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case event := <-s.out:
			if err := s.conn.Write(ctx, websocket.MessageText, mustJSON(event)); err != nil {
				return
			}
		case <-heartbeat.C:
			event := ASREvent{Type: "heartbeat", Code: "OK", SessionID: s.sessionID, Timestamp: time.Now().UnixMilli()}
			if err := s.conn.Write(ctx, websocket.MessageText, mustJSON(event)); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
func (s *asrSession) renewLease(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.leases.Renew(ctx, s.userID, s.owner); err != nil {
				s.emitError("ASR_LEASE_LOST", "speech recognition lease was lost", true)
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
func durationOr(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}
func mustJSON(value any) []byte { raw, _ := jsonMarshal(value); return raw }

// Variables keep protocol JSON isolated and replaceable in tests.
var jsonMarshal = func(value any) ([]byte, error) { return json.Marshal(value) }
var jsonError = func(raw []byte, value any) error { return json.Unmarshal(raw, value) }
