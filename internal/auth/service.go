package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"ai-meeting-go/internal/platform/httpx"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db                    *gorm.DB
	secret                []byte
	accessTTL, refreshTTL time.Duration
	now                   func() time.Time
}

func NewService(db *gorm.DB, secret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{db: db, secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL, now: time.Now}
}

type claims struct {
	TokenType string `json:"typ"`
	FamilyID  string `json:"fid,omitempty"`
	jwt.RegisteredClaims
}

func (s *Service) Register(ctx context.Context, username, password string) (User, error) {
	if len(username) < 3 || len(username) > 64 {
		return User{}, httpx.BadRequest("INVALID_USERNAME", "username length must be 3-64")
	}
	if len(password) < 8 || len(password) > 72 {
		return User{}, httpx.BadRequest("INVALID_PASSWORD", "password length must be 8-72")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	user := User{Username: username, PasswordHash: string(hash)}
	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		return User{}, httpx.Conflict("USERNAME_EXISTS", "username already exists")
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (TokenPair, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return TokenPair{}, httpx.Unauthorized("invalid username or password")
	}
	return s.issuePair(ctx, user, uuid.NewString())
}

func (s *Service) Refresh(ctx context.Context, raw string) (TokenPair, error) {
	parsed, err := s.parse(raw, "refresh")
	if err != nil {
		return TokenPair{}, httpx.Unauthorized("invalid refresh token")
	}
	hash := tokenHash(raw)
	var pair TokenPair
	var committedRejection error
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stored RefreshToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", hash).First(&stored).Error; err != nil {
			return httpx.Unauthorized("refresh token not recognized")
		}
		if stored.RevokedAt != nil {
			if err := tx.Model(&RefreshToken{}).Where("family_id = ? AND revoked_at IS NULL", stored.FamilyID).Update("revoked_at", s.now()).Error; err != nil {
				return err
			}
			// Commit family revocation, then return the authentication error outside
			// the transaction so the security update is not rolled back.
			committedRejection = httpx.Unauthorized("refresh token reuse detected")
			return nil
		}
		if s.now().After(stored.ExpiresAt) {
			return httpx.Unauthorized("refresh token expired")
		}
		uid, parseErr := strconv.ParseUint(parsed.Subject, 10, 64)
		if parseErr != nil || uid != stored.UserID {
			return httpx.Unauthorized("invalid refresh token subject")
		}
		var user User
		if err := tx.First(&user, uid).Error; err != nil {
			return httpx.Unauthorized("user not found")
		}
		var issueErr error
		pair, issueErr = s.issuePairWithDB(ctx, tx, user, stored.FamilyID)
		if issueErr != nil {
			return issueErr
		}
		now := s.now()
		result := tx.Model(&RefreshToken{}).Where("id = ? AND revoked_at IS NULL", stored.ID).Updates(map[string]any{"revoked_at": now, "replaced_by_hash": tokenHash(pair.RefreshToken)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return httpx.Unauthorized("refresh token already rotated")
		}
		return nil
	})
	if err == nil && committedRejection != nil {
		return TokenPair{}, committedRejection
	}
	return pair, err
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	parsed, err := s.parse(raw, "refresh")
	if err != nil {
		return nil
	}
	return s.db.WithContext(ctx).Model(&RefreshToken{}).Where("family_id = ? AND revoked_at IS NULL", parsed.FamilyID).Update("revoked_at", s.now()).Error
}

func (s *Service) ValidateAccess(raw string) (uint64, error) {
	c, err := s.parse(raw, "access")
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(c.Subject, 10, 64)
}

func (s *Service) issuePair(ctx context.Context, user User, family string) (TokenPair, error) {
	return s.issuePairWithDB(ctx, s.db, user, family)
}

func (s *Service) issuePairWithDB(ctx context.Context, db *gorm.DB, user User, family string) (TokenPair, error) {
	now := s.now()
	accessExp := now.Add(s.accessTTL)
	refreshExp := now.Add(s.refreshTTL)
	access, err := s.sign(user.ID, "access", "", uuid.NewString(), accessExp)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := s.sign(user.ID, "refresh", family, uuid.NewString(), refreshExp)
	if err != nil {
		return TokenPair{}, err
	}
	record := RefreshToken{UserID: user.ID, FamilyID: family, TokenHash: tokenHash(refresh), ExpiresAt: refreshExp, CreatedAt: now}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: int64(s.accessTTL.Seconds()), User: user}, nil
}

func (s *Service) sign(userID uint64, kind, family, jti string, exp time.Time) (string, error) {
	c := claims{TokenType: kind, FamilyID: family, RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.FormatUint(userID, 10), ID: jti, IssuedAt: jwt.NewNumericDate(s.now()), ExpiresAt: jwt.NewNumericDate(exp)}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
}
func (s *Service) parse(raw, expected string) (*claims, error) {
	token, err := jwt.ParseWithClaims(raw, &claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	c, ok := token.Claims.(*claims)
	if !ok || c.TokenType != expected {
		return nil, errors.New("wrong token type")
	}
	return c, nil
}
func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
