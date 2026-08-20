package worker

import (
	"context"
	"testing"

	"github.com/norest-mail/server/internal/db"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestReconcileOrphans(t *testing.T) {
	ctx := context.Background()
	mock, err := pgxmock.NewConn()
	assert.NoError(t, err)
	defer mock.Close(ctx)

	repo := db.NewMailRepository(nil)

	accountID := "acc-123"
	msgID := "<parent-msg-id>"
	canonicalThreadID := "canonical-thread"

	t.Run("merges duplicate threads", func(t *testing.T) {
		mock.ExpectBegin()
		tx, err := mock.Begin(ctx)
		assert.NoError(t, err)

		// 1. Query for orphans
		mock.ExpectQuery("SELECT DISTINCT thread_id FROM messages").
			WithArgs(accountID, msgID, canonicalThreadID).
			WillReturnRows(pgxmock.NewRows([]string{"thread_id"}).AddRow("dup-thread-1").AddRow("dup-thread-2"))

		// For dup-thread-1
		mock.ExpectExec("UPDATE messages SET thread_id").
			WithArgs(canonicalThreadID, "dup-thread-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectExec("WITH thread_messages AS").
			WithArgs(canonicalThreadID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectExec("DELETE FROM threads").
			WithArgs("dup-thread-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		// For dup-thread-2
		mock.ExpectExec("UPDATE messages SET thread_id").
			WithArgs(canonicalThreadID, "dup-thread-2").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectExec("WITH thread_messages AS").
			WithArgs(canonicalThreadID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectExec("DELETE FROM threads").
			WithArgs("dup-thread-2").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err = ReconcileOrphans(ctx, tx, repo, accountID, msgID, canonicalThreadID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
