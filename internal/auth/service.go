package auth

import (
	"context"
	"errors"
	"net/mail"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/users"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrPasswordTooWeak    = errors.New("password must be at least 8 characters long")
)

type Service struct {
	repo         *Repository
	tokenService *TokenService
}

func NewService(pool *pgxpool.Pool, jwtSecret string) *Service {
	return &Service{
		repo:         NewRepository(pool),
		tokenService: NewTokenService(jwtSecret),
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (*AuthResponse, error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}
	if len(password) < 8 {
		return nil, ErrPasswordTooWeak
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.CreateUser(ctx, email, hash)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokenService.CreateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.CreateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		ID:           user.ID,
		Email:        user.Email,
		Status:       string(user.Status),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
	}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	valid, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokenService.CreateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.CreateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		ID:           user.ID,
		Email:        user.Email,
		Status:       string(user.Status),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
	}, nil
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*users.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) ValidateToken(tokenStr string) (uuid.UUID, error) {
	return s.tokenService.ValidateToken(tokenStr)
}
