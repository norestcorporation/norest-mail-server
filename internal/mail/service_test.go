package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvisioningStatus(t *testing.T) {
	t.Run("Provisioning status structure is correct", func(t *testing.T) {
		status := &ProvisioningStatus{
			MailboxID:         "test-mailbox-id",
			AddressID:         "test-address-id",
			Status:            "provisioning",
			StalwartAccountID: nil,
			ReadyForSession:   false,
		}

		assert.Equal(t, "test-mailbox-id", status.MailboxID)
		assert.Equal(t, "test-address-id", status.AddressID)
		assert.Equal(t, "provisioning", status.Status)
		assert.Nil(t, status.StalwartAccountID)
		assert.False(t, status.ReadyForSession)
	})

	t.Run("Ready for session when active with stalwart account", func(t *testing.T) {
		stalwartID := "stalwart-account-id"
		status := &ProvisioningStatus{
			MailboxID:         "test-mailbox-id",
			AddressID:         "test-address-id",
			Status:            "active",
			StalwartAccountID: &stalwartID,
			ReadyForSession:   true,
		}

		assert.Equal(t, "active", status.Status)
		assert.NotNil(t, status.StalwartAccountID)
		assert.True(t, status.ReadyForSession)
	})

	t.Run("Not ready when status is provisioning", func(t *testing.T) {
		status := &ProvisioningStatus{
			Status:            "provisioning",
			StalwartAccountID: nil,
			ReadyForSession:   false,
		}

		assert.Equal(t, "provisioning", status.Status)
		assert.False(t, status.ReadyForSession)
	})

	t.Run("Not ready when stalwart account is missing", func(t *testing.T) {
		status := &ProvisioningStatus{
			Status:            "active",
			StalwartAccountID: nil,
			ReadyForSession:   false,
		}

		assert.Equal(t, "active", status.Status)
		assert.False(t, status.ReadyForSession)
	})
}