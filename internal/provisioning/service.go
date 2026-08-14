package provisioning

import (
	"time"

	"github.com/google/uuid"
)

type JobType string

const (
	JobTypeDomainCreate   JobType = "DOMAIN_CREATE"
	JobTypeDomainDelete   JobType = "DOMAIN_DELETE"
	JobTypeAccountCreate  JobType = "ACCOUNT_CREATE"
	JobTypeAccountDisable JobType = "ACCOUNT_DISABLE"
	JobTypeAccountEnable  JobType = "ACCOUNT_ENABLE"
	JobTypeAccountDelete  JobType = "ACCOUNT_DELETE"
)

type JobStatus string

const (
	StatusPending    JobStatus = "PENDING"
	StatusProcessing JobStatus = "PROCESSING"
	StatusSucceeded  JobStatus = "SUCCEEDED"
	StatusFailed     JobStatus = "FAILED"
	StatusRetryWait  JobStatus = "RETRY_WAIT"
)

type Job struct {
	ID            uuid.UUID  `json:"id"`
	Type          JobType    `json:"type"`
	ResourceID    uuid.UUID  `json:"resource_id"`
	Status        JobStatus  `json:"status"`
	Attempts      int        `json:"attempts"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	LastError     *string    `json:"last_error,omitempty"`
	ClaimedAt     *time.Time `json:"claimed_at,omitempty"`
	HeartbeatAt   *time.Time `json:"heartbeat_at,omitempty"`
	WorkerID      *string    `json:"worker_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
