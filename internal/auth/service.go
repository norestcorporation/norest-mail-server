package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/mail"
	"time"

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

func (s *Service) GetUserExperience(ctx context.Context, id uuid.UUID) (*WelcomeExperience, error) {
	return s.repo.GetUserExperience(ctx, id)
}

func (s *Service) CompleteWelcomeExperience(ctx context.Context, id uuid.UUID) (*WelcomeExperience, error) {
	return s.repo.CompleteWelcomeExperience(ctx, id)
}

func (s *Service) ValidateToken(tokenStr string) (uuid.UUID, error) {
	return s.tokenService.ValidateToken(tokenStr)
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResponse, error) {
	// Validate the refresh token
	userID, err := s.tokenService.ValidateToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Check if user still exists and is active
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	// Generate new tokens
	newAccessToken, err := s.tokenService.CreateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.tokenService.CreateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
	}, nil
}

func (s *Service) Logout(ctx context.Context, userID uuid.UUID) error {
	// In a more comprehensive implementation, you might want to:
	// 1. Invalidate the refresh token by storing it in a blacklist
	// 2. Remove the session from a session store
	// 3. Log the logout event for security auditing

	// For now, this is a placeholder since JWT tokens are stateless
	// The client will simply discard the tokens
	// Future enhancement: implement token blacklist in Redis/database

	return nil
}

// GenerateWebSocketTicket creates a short-lived, single-use ticket for WebSocket authentication.
func (s *Service) GenerateWebSocketTicket(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate a random 32-byte ticket
	ticketBytes := make([]byte, 32)
	if _, err := rand.Read(ticketBytes); err != nil {
		return "", err
	}
	ticket := hex.EncodeToString(ticketBytes)

	// Store ticket in database with 5-minute expiration
	query := `
		INSERT INTO websocket_tickets (ticket, user_id, expires_at)
		VALUES ($1, $2, NOW() + INTERVAL '5 minutes')
	`
	_, err := s.repo.Pool().Exec(ctx, query, ticket, userID)
	if err != nil {
		return "", err
	}

	return ticket, nil
}

// ValidateWebSocketTicket validates and consumes a WebSocket ticket.
func (s *Service) ValidateWebSocketTicket(ctx context.Context, ticket string) (uuid.UUID, error) {
	// Get and delete the ticket in a single transaction (atomic consume)
	tx, err := s.repo.Pool().Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	var expiresAt time.Time

	query := `
		SELECT user_id, expires_at
		FROM websocket_tickets
		WHERE ticket = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, query, ticket).Scan(&userID, &expiresAt)
	if err != nil {
		return uuid.Nil, errors.New("invalid ticket")
	}

	// Check if ticket is expired
	if time.Now().After(expiresAt) {
		// Delete expired ticket
		tx.Exec(ctx, "DELETE FROM websocket_tickets WHERE ticket = $1", ticket)
		return uuid.Nil, errors.New("ticket expired")
	}

	// Consume the ticket (delete it)
	_, err = tx.Exec(ctx, "DELETE FROM websocket_tickets WHERE ticket = $1", ticket)
	if err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

// WebSocketTicketHandler handles the WebSocket ticket generation endpoint
func (s *Service) WebSocketTicketHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ticket, err := s.GenerateWebSocketTicket(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to generate ticket", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ticket":"` + ticket + `"}`))
	}
}
