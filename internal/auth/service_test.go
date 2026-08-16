package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testService(t *testing.T) *Service {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &RefreshToken{}))
	return NewService(db, "01234567890123456789012345678901", time.Minute, time.Hour)
}

func TestCredentialsAccessAndLogout(t *testing.T) {
	s := testService(t)
	_, err := s.Register(t.Context(), "ab", "password123")
	require.Error(t, err)
	_, err = s.Register(t.Context(), "candidate", "short")
	require.Error(t, err)

	user, err := s.Register(t.Context(), "candidate", "password123")
	require.NoError(t, err)
	_, err = s.Register(t.Context(), "candidate", "password123")
	require.Error(t, err)
	_, err = s.Login(t.Context(), "candidate", "wrong-password")
	require.Error(t, err)

	pair, err := s.Login(t.Context(), "candidate", "password123")
	require.NoError(t, err)
	uid, err := s.ValidateAccess(pair.AccessToken)
	require.NoError(t, err)
	require.Equal(t, user.ID, uid)
	_, err = s.ValidateAccess(pair.RefreshToken)
	require.Error(t, err)
	_, err = s.ValidateAccess(pair.AccessToken + "tampered")
	require.Error(t, err)

	require.NoError(t, s.Logout(t.Context(), pair.RefreshToken))
	_, err = s.Refresh(t.Context(), pair.RefreshToken)
	require.Error(t, err)
	require.NoError(t, s.Logout(t.Context(), "not-a-token"))
}

func TestExpiredTokensAreRejected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &RefreshToken{}))
	s := NewService(db, "01234567890123456789012345678901", -time.Second, -time.Second)
	_, err = s.Register(t.Context(), "candidate", "password123")
	require.NoError(t, err)
	pair, err := s.Login(t.Context(), "candidate", "password123")
	require.NoError(t, err)
	_, err = s.ValidateAccess(pair.AccessToken)
	require.Error(t, err)
	_, err = s.Refresh(t.Context(), pair.RefreshToken)
	require.Error(t, err)
}
func TestRefreshRotationAndReuseRevokesFamily(t *testing.T) {
	s := testService(t)
	u, err := s.Register(t.Context(), "candidate", "password123")
	require.NoError(t, err)
	first, err := s.Login(t.Context(), u.Username, "password123")
	require.NoError(t, err)
	second, err := s.Refresh(t.Context(), first.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, first.RefreshToken, second.RefreshToken)
	_, err = s.Refresh(t.Context(), first.RefreshToken)
	require.Error(t, err)
	_, err = s.Refresh(t.Context(), second.RefreshToken)
	require.Error(t, err)
}
