package mail

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/norest-mail/server/internal/stalwart"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestThreadResolver(t *testing.T) {
	ctx := context.Background()
	accountID := "acc-123"
	now := time.Now()

	t.Run("existing message", func(t *testing.T) {
		mock, err := pgxmock.NewConn()
		assert.NoError(t, err)
		defer mock.Close(ctx)
		mock.ExpectBegin()
		tx, err := mock.Begin(ctx)
		assert.NoError(t, err)

		email := &stalwart.Email{
			ID: "email-1",
		}

		mock.ExpectQuery("SELECT thread_id FROM messages").
			WithArgs(accountID, "email-1").
			WillReturnRows(pgxmock.NewRows([]string{"thread_id"}).AddRow("thread-1"))

		res, err := ResolveThread(ctx, tx, accountID, email, &now)
		assert.NoError(t, err)
		assert.Equal(t, "thread-1", res.ThreadID)
		assert.False(t, res.IsNew)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("in-reply-to matches", func(t *testing.T) {
		mock, err := pgxmock.NewConn()
		assert.NoError(t, err)
		defer mock.Close(ctx)
		mock.ExpectBegin()
		tx, err := mock.Begin(ctx)
		assert.NoError(t, err)

		email := &stalwart.Email{
			ID:        "email-2",
			InReplyTo: []string{"<msg-id-1>"},
		}

		mock.ExpectQuery("SELECT thread_id FROM messages WHERE account_id = \\$1 AND stalwart_email_id = \\$2").
			WithArgs(accountID, "email-2").
			WillReturnError(pgx.ErrNoRows)

		mock.ExpectQuery("SELECT thread_id FROM messages WHERE account_id = \\$1 AND message_id = ANY\\(\\$2\\) LIMIT 1").
			WithArgs(accountID, []string{"<msg-id-1>"}).
			WillReturnRows(pgxmock.NewRows([]string{"thread_id"}).AddRow("thread-2"))

		res, err := ResolveThread(ctx, tx, accountID, email, &now)
		assert.NoError(t, err)
		assert.Equal(t, "thread-2", res.ThreadID)
		assert.False(t, res.IsNew)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("new thread fallback", func(t *testing.T) {
		mock, err := pgxmock.NewConn()
		assert.NoError(t, err)
		defer mock.Close(ctx)
		mock.ExpectBegin()
		tx, err := mock.Begin(ctx)
		assert.NoError(t, err)

		email := &stalwart.Email{
			ID:      "email-3",
			Subject: "Hello World",
		}

		mock.ExpectQuery("SELECT thread_id FROM messages WHERE account_id = \\$1 AND stalwart_email_id = \\$2").
			WithArgs(accountID, "email-3").
			WillReturnError(pgx.ErrNoRows)

		// Subject match is skipped because it's not a reply subject

		mock.ExpectQuery("INSERT INTO threads").
			WithArgs(accountID, "hello world").
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("new-thread-1"))

		res, err := ResolveThread(ctx, tx, accountID, email, &now)
		assert.NoError(t, err)
		assert.Equal(t, "new-thread-1", res.ThreadID)
		assert.True(t, res.IsNew)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
