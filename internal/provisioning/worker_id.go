package provisioning

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// GenerateWorkerID creates a unique worker identifier.
// If WORKER_ID is set in environment, use it.
// Otherwise, generate a runtime-safe identifier.
func GenerateWorkerID() string {
	if envID := os.Getenv("WORKER_ID"); envID != "" {
		return envID
	}
	// Generate a unique ID with timestamp for uniqueness across restarts
	return fmt.Sprintf("worker-%s-%d", uuid.New().String()[:8], time.Now().Unix())
}
