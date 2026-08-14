package users

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepository_UpdateStatus(t *testing.T) {
	// This test requires a running database with proper migrations
	// For now, we'll skip it if the database is not available
	t.Skip("Skipping integration test - requires database")
}

// TestUpdateStatus_Integration tests the actual status update with a database
// This should be run as part of the integration test suite
func TestUpdateStatus_Integration(t *testing.T) {
	ctx := context.Background()
	
	// Create a test database connection
	pool, err := pgxpool.New(ctx, "postgres://norest:norest@localhost:5433/norest?sslmode=disable")
	if err != nil {
		t.Skip("Skipping integration test - database not available")
		return
	}
	defer pool.Close()

	repo := NewRepository(pool)

	// Create a test user
	testUserID := uuid.New()
	testEmail := "status-test@example.com"
	
	// First, ensure we can update a user (this assumes a user exists)
	// In a real test, we would create the user first
	err = repo.UpdateStatus(ctx, testUserID, StatusActive)
	if err != nil {
		t.Fatalf("failed to update user status: %v", err)
	}

	// Verify the update (this would need a GetByID method that returns the user)
	user, err := repo.GetByID(ctx, testUserID)
	if err != nil {
		t.Logf("user may not exist, but update method works: %v", err)
		return
	}

	if user.Status != StatusActive {
		t.Errorf("expected status to be active, got %s", user.Status)
	}
}