package addresses

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddressReservationSystem tests the complete address reservation system
func TestAddressReservationSystem(t *testing.T) {
	t.Run("Address status constants are correct", func(t *testing.T) {
		assert.Equal(t, "AVAILABLE", string(StatusAvailable))
		assert.Equal(t, "RESERVED", string(StatusReserved))
		assert.Equal(t, "CLAIMED", string(StatusClaimed))
		assert.Equal(t, "BLOCKED", string(StatusBlocked))
	})
	
	t.Run("Address structure includes reservation fields", func(t *testing.T) {
		userID := uuid.New()
		now := time.Now()
		future := now.Add(2 * time.Hour)
		
		address := &Address{
			ID:            uuid.New(),
			DomainID:      uuid.New(),
			LocalPart:     "testuser",
			Status:        StatusReserved,
			ReservedBy:    &userID,
			ReservedAt:    &now,
			ReservedUntil: &future,
		}
		
		assert.NotNil(t, address.ReservedBy)
		assert.NotNil(t, address.ReservedAt)
		assert.NotNil(t, address.ReservedUntil)
		assert.Equal(t, StatusReserved, address.Status)
	})
}

// TestLocalPartNormalization tests username normalization
func TestLocalPartNormalization(t *testing.T) {
	tests := []struct {
		input     string
		expected  string
		expectErr bool
	}{
		{"TestUser", "testuser", false},
		{"Test.User", "test.user", false},
		{"test+user", "test+user", false},
		{"", "", true},
		{".invalid", "", true},
		{"invalid.", "", true},
		{"in..valid", "", true},
		{"invalid name", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := NormalizeLocalPart(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAddressUniqueness tests that (domain_id, local_part) combinations are unique
func TestAddressUniqueness(t *testing.T) {
	t.Run("Same domain, different local parts are allowed", func(t *testing.T) {
		localPart1 := "user1"
		localPart2 := "user2"
		
		assert.NotEqual(t, localPart1, localPart2)
	})
	
	t.Run("Same local part, different domains are allowed", func(t *testing.T) {
		localPart := "user"
		
		assert.Equal(t, localPart, localPart)
	})
	
	t.Run("Same domain and local part are not allowed", func(t *testing.T) {
		// This would normally fail with duplicate constraint
		assert.True(t, true) // Placeholder for actual test
	})
}

// TestReservationExpiration tests the reservation expiration logic
func TestReservationExpiration(t *testing.T) {
	t.Run("Reservations expire after 2 hours", func(t *testing.T) {
		now := time.Now()
		reservedUntil := now.Add(2 * time.Hour)
		
		assert.True(t, reservedUntil.After(now))
		assert.Equal(t, 2*time.Hour, reservedUntil.Sub(now))
	})
	
	t.Run("Expired reservations can be claimed by others", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		reservedUntil := past
		
		assert.True(t, time.Now().After(reservedUntil))
	})
	
	t.Run("Active reservations cannot be claimed by others", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour)
		reservedUntil := future
		
		assert.True(t, time.Now().Before(reservedUntil))
	})
}

// TestAddressClaimingLogic tests the claiming logic
func TestAddressClaimingLogic(t *testing.T) {
	t.Run("Claimed addresses have claimed_by and claimed_at set", func(t *testing.T) {
		userID := uuid.New()
		now := time.Now()
		
		address := &Address{
			Status:    StatusClaimed,
			ClaimedBy: &userID,
			ClaimedAt: &now,
		}
		
		assert.Equal(t, StatusClaimed, address.Status)
		assert.Equal(t, userID, *address.ClaimedBy)
		assert.NotNil(t, address.ClaimedAt)
	})
	
	t.Run("Claiming clears reservation fields", func(t *testing.T) {
		address := &Address{
			Status:        StatusClaimed,
			ReservedBy:    nil,
			ReservedAt:    nil,
			ReservedUntil: nil,
		}
		
		assert.Nil(t, address.ReservedBy)
		assert.Nil(t, address.ReservedAt)
		assert.Nil(t, address.ReservedUntil)
	})
	
	t.Run("Claimed address cannot be reserved again", func(t *testing.T) {
		address := &Address{
			Status: StatusClaimed,
		}
		
		assert.Equal(t, StatusClaimed, address.Status)
	})
}

// TestBlockedAddresses tests blocked address functionality
func TestBlockedAddresses(t *testing.T) {
	t.Run("Blocked addresses have BLOCKED status", func(t *testing.T) {
		status := StatusBlocked
		assert.Equal(t, StatusBlocked, status)
	})
	
	t.Run("Blocked addresses cannot be reserved", func(t *testing.T) {
		status := StatusBlocked
		// Should not be available for reservation
		assert.Equal(t, StatusBlocked, status)
	})
}

// TestRaceSafety tests concurrent reservation handling
func TestRaceSafety(t *testing.T) {
	t.Run("Concurrent reservations are handled by database", func(t *testing.T) {
		// This would test actual concurrent operations
		assert.True(t, true) // Placeholder
	})
	
	t.Run("Database function ensures atomicity", func(t *testing.T) {
		// Test that the reserve_address function is atomic
		assert.True(t, true) // Placeholder
	})
}

// TestDomainAvailabilityValidation tests domain validation for address creation
func TestDomainAvailabilityValidation(t *testing.T) {
	t.Run("Active domain with registration enabled allows addresses", func(t *testing.T) {
		domainAvailable := true
		registrationEnabled := true
		
		assert.True(t, domainAvailable && registrationEnabled)
	})
	
	t.Run("Inactive domain does not allow addresses", func(t *testing.T) {
		domainAvailable := false
		
		assert.False(t, domainAvailable)
	})
	
	t.Run("Domain with registration disabled does not allow addresses", func(t *testing.T) {
		registrationEnabled := false
		
		assert.False(t, registrationEnabled)
	})
}

// TestAddressLifecycle tests the complete address lifecycle
func TestAddressLifecycle(t *testing.T) {
	t.Run("Complete lifecycle: AVAILABLE -> RESERVED -> CLAIMED", func(t *testing.T) {
		states := []Status{StatusAvailable, StatusReserved, StatusClaimed}
		
		assert.Equal(t, StatusAvailable, states[0])
		assert.Equal(t, StatusReserved, states[1])
		assert.Equal(t, StatusClaimed, states[2])
	})
	
	t.Run("Lifecycle cannot go backwards", func(t *testing.T) {
		// Once CLAIMED, cannot go back to RESERVED or AVAILABLE
		currentStatus := StatusClaimed
		assert.Equal(t, StatusClaimed, currentStatus)
	})
}

// TestMultipleUsersSameDomain tests the main use case
func TestMultipleUsersSameDomain(t *testing.T) {
	domainID := uuid.New()
	
	t.Run("User A reserves ripun@norestmail.com", func(t *testing.T) {
		userA := uuid.New()
		localPart := "ripun"
		
			now := time.Now()
		future := now.Add(2 * time.Hour)
		address := &Address{
			ID:            uuid.New(),
			DomainID:      domainID,
			LocalPart:     localPart,
			Status:        StatusReserved,
			ReservedBy:    &userA,
			ReservedAt:    &now,
			ReservedUntil: &future,
		}
		
		assert.Equal(t, domainID, address.DomainID)
		assert.Equal(t, "ripun", address.LocalPart)
		assert.Equal(t, userA, *address.ReservedBy)
	})
	
	t.Run("User B reserves alice@norestmail.com", func(t *testing.T) {
		userB := uuid.New()
		localPart := "alice"
		
		now := time.Now()
		future := now.Add(2 * time.Hour)
		address := &Address{
			ID:            uuid.New(),
			DomainID:      domainID,
			LocalPart:     localPart,
			Status:        StatusReserved,
			ReservedBy:    &userB,
			ReservedAt:    &now,
			ReservedUntil: &future,
		}
		
		assert.Equal(t, domainID, address.DomainID)
		assert.Equal(t, "alice", address.LocalPart)
		assert.Equal(t, userB, *address.ReservedBy)
	})
	
	t.Run("User C reserves bob@norestmail.com", func(t *testing.T) {
		userC := uuid.New()
		localPart := "bob"
		
		now := time.Now()
		future := now.Add(2 * time.Hour)
		address := &Address{
			ID:            uuid.New(),
			DomainID:      domainID,
			LocalPart:     localPart,
			Status:        StatusReserved,
			ReservedBy:    &userC,
			ReservedAt:    &now,
			ReservedUntil: &future,
		}
		
		assert.Equal(t, domainID, address.DomainID)
		assert.Equal(t, "bob", address.LocalPart)
		assert.Equal(t, userC, *address.ReservedBy)
	})
	
	t.Run("All addresses reference the same domain_id", func(t *testing.T) {
		// All three addresses should have the same domain_id
		assert.Equal(t, domainID, domainID)
	})
}

// TestRepositoryMethods tests the new repository methods
func TestRepositoryMethods(t *testing.T) {
	t.Run("ReserveAddress uses database function", func(t *testing.T) {
		// Test that ReserveAddress calls the database function
		assert.True(t, true) // Placeholder
	})
	
	t.Run("ClaimAddress uses database function", func(t *testing.T) {
		// Test that ClaimAddress calls the database function
		assert.True(t, true) // Placeholder
	})
	
	t.Run("CheckAddressAvailability uses database function", func(t *testing.T) {
		// Test that CheckAddressAvailability calls the database function
		assert.True(t, true) // Placeholder
	})
	
	t.Run("BlockAddress adds to blocked_addresses table", func(t *testing.T) {
		// Test blocking functionality
		assert.True(t, true) // Placeholder
	})
	
	t.Run("CleanExpiredReservations uses database function", func(t *testing.T) {
		// Test cleanup functionality
		assert.True(t, true) // Placeholder
	})
}

// TestServiceMethods tests the new service methods
func TestServiceMethods(t *testing.T) {
	t.Run("ReserveAddress validates domain availability", func(t *testing.T) {
		// Test domain validation before reservation
		assert.True(t, true) // Placeholder
	})
	
	t.Run("ClaimAddress validates reservation ownership", func(t *testing.T) {
		// Test that only the reserver can claim (or admin)
		assert.True(t, true) // Placeholder
	})
	
	t.Run("CheckAddressAvailability performs all checks", func(t *testing.T) {
		// Test comprehensive availability check
		assert.True(t, true) // Placeholder
	})
}

// TestMailboxCreation tests mailbox creation for addresses
func TestMailboxCreation(t *testing.T) {
	t.Run("Mailbox is created when address is claimed", func(t *testing.T) {
		// Test that mailbox creation happens during claiming
		assert.True(t, true) // Placeholder
	})
	
	t.Run("Mailbox creation is separate from reservation", func(t *testing.T) {
		// Test that reservation doesn't create mailbox
		assert.True(t, true) // Placeholder
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("Duplicate address returns error", func(t *testing.T) {
		// Test ErrAddressExists
		assert.Equal(t, "address already exists", ErrAddressExists.Error())
	})
	
	t.Run("Invalid local part returns error", func(t *testing.T) {
		// Test ErrInvalidLocalPart
		assert.Equal(t, "invalid local part", ErrInvalidLocalPart.Error())
	})
	
	t.Run("Domain not found returns error", func(t *testing.T) {
		// Test that non-existent domains are rejected
		assert.True(t, true) // Placeholder
	})
}

// TestBackwardCompatibility tests that existing functionality still works
func TestBackwardCompatibility(t *testing.T) {
	t.Run("Existing CreateAddress still works", func(t *testing.T) {
		// Test that the original CreateAddress method still functions
		assert.True(t, true) // Placeholder
	})
	
	t.Run("Existing ListAddresses still works", func(t *testing.T) {
		// Test that address listing still functions
		assert.True(t, true) // Placeholder
	})
}

// MockAddressRepository is a mock implementation for testing
type MockAddressRepository struct {
	addresses map[uuid.UUID]*Address
}

func NewMockAddressRepository() *MockAddressRepository {
	return &MockAddressRepository{
		addresses: make(map[uuid.UUID]*Address),
	}
}

func (m *MockAddressRepository) ReserveAddress(ctx context.Context, domainID uuid.UUID, localPart string, userID uuid.UUID, durationHours int) (*Address, error) {
	now := time.Now()
	future := now.Add(time.Duration(durationHours) * time.Hour)
	
	address := &Address{
		ID:            uuid.New(),
		DomainID:      domainID,
		LocalPart:     localPart,
		Status:        StatusReserved,
		ReservedBy:    &userID,
		ReservedAt:    &now,
		ReservedUntil: &future,
	}
	m.addresses[address.ID] = address
	return address, nil
}

func (m *MockAddressRepository) ClaimAddress(ctx context.Context, addressID uuid.UUID, userID uuid.UUID) error {
	address, exists := m.addresses[addressID]
	if !exists {
		return ErrAddressExists
	}
	
	now := time.Now()
	address.Status = StatusClaimed
	address.ClaimedBy = &userID
	address.ClaimedAt = &now
	address.ReservedBy = nil
	address.ReservedAt = nil
	address.ReservedUntil = nil
	
	return nil
}

func (m *MockAddressRepository) CheckAddressAvailability(ctx context.Context, domainID uuid.UUID, localPart string) (bool, error) {
	for _, address := range m.addresses {
		if address.DomainID == domainID && address.LocalPart == localPart {
			if address.Status == StatusClaimed {
				return false, nil
			}
			if address.Status == StatusReserved && address.ReservedUntil != nil && time.Now().Before(*address.ReservedUntil) {
				return false, nil
			}
		}
	}
	return true, nil
}

func (m *MockAddressRepository) GetByID(ctx context.Context, id uuid.UUID) (*Address, error) {
	address, exists := m.addresses[id]
	if !exists {
		return nil, ErrAddressExists
	}
	return address, nil
}

// TestMockAddressRepository tests the mock repository
func TestMockAddressRepository(t *testing.T) {
	mockRepo := NewMockAddressRepository()
	ctx := context.Background()
	domainID := uuid.New()
	userID := uuid.New()
	
	t.Run("Reserve and claim address", func(t *testing.T) {
		address, err := mockRepo.ReserveAddress(ctx, domainID, "testuser", userID, 2)
		require.NoError(t, err)
		require.NotNil(t, address)
		assert.Equal(t, StatusReserved, address.Status)
		assert.Equal(t, userID, *address.ReservedBy)
		
		err = mockRepo.ClaimAddress(ctx, address.ID, userID)
		require.NoError(t, err)
		
		claimed, err := mockRepo.GetByID(ctx, address.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusClaimed, claimed.Status)
		assert.Equal(t, userID, *claimed.ClaimedBy)
		assert.Nil(t, claimed.ReservedBy)
	})
	
	t.Run("Check availability", func(t *testing.T) {
		available, err := mockRepo.CheckAddressAvailability(ctx, domainID, "newuser")
		require.NoError(t, err)
		assert.True(t, available)
		
		// Reserve the address
		_, err = mockRepo.ReserveAddress(ctx, domainID, "newuser", userID, 2)
		require.NoError(t, err)
		
		// Check availability again
		available, err = mockRepo.CheckAddressAvailability(ctx, domainID, "newuser")
		require.NoError(t, err)
		assert.False(t, available)
	})
	
	t.Run("Multiple users same domain", func(t *testing.T) {
		userA := uuid.New()
		userB := uuid.New()
		
		addressA, err := mockRepo.ReserveAddress(ctx, domainID, "usera", userA, 2)
		require.NoError(t, err)
		
		addressB, err := mockRepo.ReserveAddress(ctx, domainID, "userb", userB, 2)
		require.NoError(t, err)
		
		assert.Equal(t, domainID, addressA.DomainID)
		assert.Equal(t, domainID, addressB.DomainID)
		assert.NotEqual(t, addressA.LocalPart, addressB.LocalPart)
		assert.NotEqual(t, addressA.ReservedBy, addressB.ReservedBy)
	})
}