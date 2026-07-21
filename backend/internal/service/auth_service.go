package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/joythejaks/palmyield/backend/internal/auth"
	"github.com/joythejaks/palmyield/backend/internal/repository"
	"github.com/joythejaks/palmyield/backend/internal/repository/sqlcgen"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrAccountDisabled     = errors.New("account disabled")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

type AuthService struct {
	repo       *repository.Repository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(repo *repository.Repository, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{repo: repo, jwtSecret: jwtSecret, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Login(ctx context.Context, identifier, password string) (TokenPair, error) {
	user, err := s.repo.GetUserByIdentifier(ctx, pgtype.Text{String: identifier, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, err
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return TokenPair{}, ErrInvalidCredentials
	}
	if user.Status == "disabled" {
		return TokenPair{}, ErrAccountDisabled
	}
	if user.Status == "invited" {
		user, err = s.repo.UpdateUserStatus(ctx, sqlcgen.UpdateUserStatusParams{ID: user.ID, Status: "active"})
		if err != nil {
			return TokenPair{}, err
		}
	}
	return s.issueTokenPair(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	hash := auth.HashRefreshToken(refreshToken)
	stored, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, err
	}
	if stored.RevokedAt.Valid || stored.ExpiresAt.Time.Before(time.Now()) {
		return TokenPair{}, ErrInvalidRefreshToken
	}
	// Atomic compare-and-set: if a concurrent request already revoked this
	// token between our SELECT and this UPDATE, rowsAffected is 0 and we
	// reject rather than mint a second token pair from the same token.
	rowsAffected, err := s.repo.RevokeRefreshToken(ctx, stored.ID)
	if err != nil {
		return TokenPair{}, err
	}
	if rowsAffected == 0 {
		return TokenPair{}, ErrInvalidRefreshToken
	}
	user, err := s.repo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		return TokenPair{}, err
	}
	return s.issueTokenPair(ctx, user)
}

// Logout is idempotent: an already-revoked or unknown refresh token is not an error.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := auth.HashRefreshToken(refreshToken)
	stored, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	_, err = s.repo.RevokeRefreshToken(ctx, stored.ID)
	return err
}

type InviteResult struct {
	UserID       string
	TempPassword string
}

// Invite creates an admin- or farmer-role user scoped to the inviting
// admin's own cooperative. The temp password is returned once for the
// admin to relay out-of-band; there is no email/SMS delivery yet.
func (s *AuthService) Invite(ctx context.Context, cooperativeID, email, phone, role string) (InviteResult, error) {
	coopID, err := uuid.Parse(cooperativeID)
	if err != nil {
		return InviteResult{}, err
	}

	tempPassword, err := auth.GenerateTempPassword(12)
	if err != nil {
		return InviteResult{}, err
	}
	hash, err := auth.HashPassword(tempPassword)
	if err != nil {
		return InviteResult{}, err
	}

	user, err := s.repo.CreateUser(ctx, sqlcgen.CreateUserParams{
		CooperativeID: pgtype.UUID{Bytes: coopID, Valid: true},
		Email:         toText(email),
		Phone:         toText(phone),
		PasswordHash:  hash,
		Role:          role,
		Status:        "invited",
	})
	if err != nil {
		return InviteResult{}, err
	}

	return InviteResult{UserID: uuid.UUID(user.ID.Bytes).String(), TempPassword: tempPassword}, nil
}

func (s *AuthService) issueTokenPair(ctx context.Context, user sqlcgen.User) (TokenPair, error) {
	userID := uuid.UUID(user.ID.Bytes).String()
	coopID := uuid.UUID(user.CooperativeID.Bytes).String()

	access, err := auth.GenerateAccessToken(s.jwtSecret, userID, coopID, user.Role, s.accessTTL)
	if err != nil {
		return TokenPair{}, err
	}

	refreshPlain, refreshHash, err := auth.GenerateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	if _, err := s.repo.CreateRefreshToken(ctx, sqlcgen.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(s.refreshTTL), Valid: true},
	}); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: access, RefreshToken: refreshPlain}, nil
}

func toText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
