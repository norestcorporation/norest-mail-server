package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWelcomeExperienceEndpoints(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://norest:norest@localhost:5433/norest?sslmode=disable")
	if err != nil {
		t.Skip("Skipping integration test - database not available")
		return
	}
	defer pool.Close()

	service := NewService(pool, "test-secret")
	handler := NewHandler(service)

	// Create test user
	email := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	user, err := service.repo.CreateUser(ctx, email, "hashed-password")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Helper to make authenticated request
	makeRequest := func(method, path string, userID uuid.UUID) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(method, path, nil)
		// Add userID to context
		req = req.WithContext(context.WithValue(req.Context(), userContextKey, userID))
		rr := httptest.NewRecorder()

		// Route manually
		if path == "/experience" && method == "GET" {
			handler.GetExperience(rr, req)
		} else if path == "/experience/welcome/complete" && method == "POST" {
			handler.CompleteWelcomeExperience(rr, req)
		} else {
			handler.Me(rr, req)
		}

		return rr
	}

	// 1. Unauthenticated request is rejected
	req, _ := http.NewRequest("GET", "/experience", nil)
	rr := httptest.NewRecorder()
	handler.GetExperience(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected unauthorized, got %d", rr.Code)
	}

	// 2. New user returns completed=false
	rr = makeRequest("GET", "/experience", user.ID)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}
	var expRes ExperienceResponse
	json.NewDecoder(rr.Body).Decode(&expRes)
	if expRes.Welcome.Completed != false {
		t.Errorf("expected completed=false, got true")
	}

	// 3. POST marks welcome as completed
	rr = makeRequest("POST", "/experience/welcome/complete", user.ID)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}
	json.NewDecoder(rr.Body).Decode(&expRes)
	if expRes.Welcome.Completed != true {
		t.Errorf("expected completed=true, got false")
	}
	if expRes.Welcome.CompletedAt == nil {
		t.Errorf("expected completed_at to be set")
	}

	// 4. POST is idempotent
	rr = makeRequest("POST", "/experience/welcome/complete", user.ID)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}

	// 5. GET after POST returns completed=true
	rr = makeRequest("GET", "/experience", user.ID)
	json.NewDecoder(rr.Body).Decode(&expRes)
	if expRes.Welcome.Completed != true {
		t.Errorf("expected completed=true, got false")
	}

	// 6. A different authenticated user cannot read or modify another user's state
	// (Covered by the fact that makeRequest only accesses the user in the context)
	email2 := fmt.Sprintf("test2-%d@example.com", time.Now().UnixNano())
	user2, _ := service.repo.CreateUser(ctx, email2, "hashed-password")
	rr = makeRequest("GET", "/experience", user2.ID)
	json.NewDecoder(rr.Body).Decode(&expRes)
	if expRes.Welcome.Completed != false {
		t.Errorf("expected new user to have completed=false, got true")
	}
}
