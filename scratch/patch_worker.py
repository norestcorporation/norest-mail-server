import re

with open("internal/mail/worker.go", "r") as f:
    content = f.read()

# Replace reconcileStaleIntents
orig_func = """func (w *ReconciliationWorker) reconcileStaleIntents(ctx context.Context) {
// For now, this just marks them FAILED to prevent them from hanging forever.
// A true reconciliation would query JMAP to see if the intent actually succeeded.
// We'll keep it simple for this milestone.
query := `
ciliation_logs
= 'FAILED', updated_at = NOW(), stalwart_response = '{"error": "reconciliation_timeout"}'
IN ('PENDING', 'EXECUTING', 'UNKNOWN')
D updated_at < NOW() - INTERVAL '5 minutes'
`
_, err := w.pool.Exec(ctx, query)
if err != nil {
reconcile stale intents", "error", err)
}
}"""

new_func = """func (w *ReconciliationWorker) reconcileStaleIntents(ctx context.Context) {
query := `
user_id, action, intent_payload
ciliation_logs
IN ('PENDING', 'EXECUTING', 'UNKNOWN')
D updated_at < NOW() - INTERVAL '10 seconds'
 created_at ASC
UPDATE SKIP LOCKED
`
rows, err := w.pool.Query(ctx, query)
if err != nil {
fetch stale intents", "error", err)

}
defer rows.Close()

type intentRec struct {
    string
string
  string
[]byte
}
var records []intentRec
for rows.Next() {
intentRec
:= rows.Scan(&r.ID, &r.UserID, &r.Action, &r.Payload); err == nil {
append(records, r)
rec := range records {
tent(ctx, rec.ID, rec.UserID, rec.Action, rec.Payload)
}
}

func (w *ReconciliationWorker) processStaleIntent(ctx context.Context, logID, userID, action string, payload []byte) {
// Get Stalwart account ID
var stalwartAcctID string
err := w.pool.QueryRow(ctx, `
t_id FROM addresses a
 mailboxes m ON m.id = a.mailbox_id
= $1 AND a.is_primary = true
`, userID).Scan(&stalwartAcctID)

if err != nil {
mail_reconciliation_logs SET status = 'FAILED' WHERE id = $1", logID)

}

var parsed map[string]any
json.Unmarshal(payload, &parsed)

msgID, _ := parsed["message_id"].(string)
if msgID == "" {
= parsed["email_id"].(string)
}
if msgID == "" {
= parsed["draft_id"].(string)
}

success := false

if msgID != "" {
if message exists and matches intent
g, err := w.stalwartClient.EmailGet(ctx, stalwartAcctID, []string{msgID}, []string{"id", "keywords", "mailboxIds"})
== nil && len(existing.List) > 0 {
true
add deeper checks for move/keyword here based on action
if err == nil && action == "message.deleted" {
true // It's deleted, which is what we wanted
{
event for real success
:= w.pool.Begin(ctx)
== nil {
t(ctx, tx, userID, action, parsed)
mail_reconciliation_logs SET status = 'SUCCESS', updated_at = NOW() WHERE id = $1", logID)
{
 failed in Stalwart
mail_reconciliation_logs SET status = 'FAILED', updated_at = NOW() WHERE id = $1", logID)
}
}"""

content = content.replace(orig_func, new_func)

with open("internal/mail/worker.go", "w") as f:
    f.write(content)
