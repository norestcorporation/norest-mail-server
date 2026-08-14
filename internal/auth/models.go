package auth

import "github.com/google/uuid"

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Status       string    `json:"status"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
}

type UserResponse struct {
	ID     uuid.UUID `json:"id"`
	Email  string    `json:"email"`
	Status string    `json:"status"`
}
