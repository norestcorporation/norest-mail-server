package product

import (
	"time"

	"github.com/google/uuid"
)

type AccountStatus string

const (
	StatusActive    AccountStatus = "ACTIVE"
	StatusSuspended AccountStatus = "SUSPENDED"
	StatusDisabled  AccountStatus = "DISABLED"
	StatusPending   AccountStatus = "PENDING"
)

type Account struct {
	ID        uuid.UUID
	Status    AccountStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Plan struct {
	ID              uuid.UUID
	Code            string
	Name            string
	Description     *string
	Status          string
	MaxDomains      int
	MaxMailboxes    int
	MaxAddresses    int
	MaxStorageBytes int64
	Features        map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type SubscriptionStatus string

const (
	SubStatusTrialing SubscriptionStatus = "TRIALING"
	SubStatusActive   SubscriptionStatus = "ACTIVE"
	SubStatusPastDue  SubscriptionStatus = "PAST_DUE"
	SubStatusCanceled SubscriptionStatus = "CANCELED"
	SubStatusExpired  SubscriptionStatus = "EXPIRED"
	SubStatusPaused   SubscriptionStatus = "PAISED"
)

type Subscription struct {
	ID                     uuid.UUID
	ProductAccountID       uuid.UUID
	PlanID                 uuid.UUID
	Status                 SubscriptionStatus
	Provider               string
	ProviderCustomerID     *string
	ProviderSubscriptionID *string
	CurrentPeriodStart     time.Time
	CurrentPeriodEnd       time.Time
	CancelAtPeriodEnd      bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
