package db

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestUpdateThreadProjection(t *testing.T) {
	ctx := context.Background()
	mock, err := pgxmock.NewConn()
	assert.NoError(t, err)
	defer mock.Close(ctx)

	repo := NewMailRepository(nil) // we won't use the pool, just the Tx

	t.Run("UpdateThreadProjection executes correctly", func(t *testing.T) {
		mock.ExpectBegin()
		tx, err := mock.Begin(ctx)
		assert.NoError(t, err)

		threadID := "thread-123"

		// We expect the Exec call for the projection calculation
		mock.ExpectExec("WITH thread_messages AS").
			WithArgs(threadID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err = repo.UpdateThreadProjection(ctx, tx, threadID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
