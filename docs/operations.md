# Norest Mail Operations Guide

This guide covers operational procedures for running Norest Mail in production.

## Startup

1. Start the services:
   ```bash
   docker compose up -d
   ```

2. Verify all services are healthy:
   ```bash
   docker compose ps
   curl http://localhost:8080/health
   curl http://localhost:8080/health/ready
   ```

## Shutdown

1. Graceful shutdown:
   ```bash
   docker compose down
   ```

2. For production with zero downtime, use rolling restarts across instances.

## Readiness

The API is ready when:
- `/health/live` returns 200 (process is alive)
- `/health/ready` returns 200 (dependencies are healthy)
- `/health/db` returns 200 (database is accessible)
- `/health/stalwart` returns 200 (Stalwart is accessible)

## Worker Backlog

Monitor the provisioning backlog:
```bash
curl http://localhost:8080/metrics | jq '{
  pending_jobs,
  processing_jobs,
  retry_jobs,
  failed_jobs,
  oldest_pending_job
}'
```

- `pending_jobs`: Jobs waiting to be processed
- `processing_jobs`: Jobs currently being processed
- `retry_jobs`: Jobs waiting for retry
- `failed_jobs`: Jobs that have permanently failed
- `oldest_pending_job`: Age of the oldest pending job

## Failed Jobs

When jobs fail:
1. Check the worker logs for error details
2. Look for `error_code` in the logs to classify the error type
3. Check `/metrics` for `jobs_failed` count
4. For permanent failures, investigate the root cause and retry after fixing

Common error codes:
- `TEMPORARY`: Transient errors (will retry)
- `TIMEOUT`: Operation timed out (will retry)
- `RATE_LIMITED`: External API rate limit (will retry)
- `PERMANENT`: Fatal errors (manual intervention required)

## Common Errors

### Database Unavailable
- Symptom: `/health/db` returns 503
- Action: Check PostgreSQL health, restart if needed
- Recovery: Automatic when database comes back

### Stalwart Unavailable
- Symptom: `/health/stalwart` returns 503
- Action: Check Stalwart health, restart if needed
- Recovery: Automatic when Stalwart comes back

### Worker Not Processing Jobs
- Symptom: `pending_jobs` increasing, `processing_jobs` at 0
- Action: Check worker logs, restart worker if stuck
- Recovery: Jobs will be reclaimed automatically after lease expiry

### High Error Rate
- Symptom: `jobs_failed` increasing rapidly
- Action: Check logs for error patterns, fix root cause
- Recovery: Fix underlying issue, then process failed jobs manually if needed

## Restarting Services

### API Server
```bash
docker compose restart norest-api
```

### Worker
```bash
docker compose restart norest-worker
```

### PostgreSQL
```bash
docker compose restart postgres
```

### Stalwart
```bash
docker compose restart stalwart
```

## Checking Stalwart

1. Check Stalwart health:
   ```bash
   curl http://localhost:8081/admin
   ```

2. Check Stalwart JMAP:
   ```bash
   curl http://localhost:8081/.well-known/jmap
   ```

3. Verify domains in Stalwart:
   ```bash
   curl -u admin:change-me-development-only \
     http://localhost:8081/jmap \
     -H "Content-Type: application/json" \
     -d '{"using":["urn:ietf:params:jmap:core","urn:stalwart:jmap"],"methodCalls":[["x:Domain/get",{},"0"]]}'
   ```

## Checking PostgreSQL

1. Check PostgreSQL health:
   ```bash
   docker compose exec postgres pg_isready -U norest -d norest
   ```

2. Check connection pool:
   ```bash
   curl http://localhost:8080/metrics | jq '.db_pool_utilization'
   ```

3. Check database errors:
   ```bash
   curl http://localhost:8080/metrics | jq '.db_errors'
   ```

## Checking Provisioning

1. Check worker status:
   ```bash
   docker compose logs norest-worker --tail 50
   ```

2. Check job statistics:
   ```bash
   curl http://localhost:8080/metrics | jq '{
     jobs_started,
     jobs_succeeded,
     jobs_failed,
     jobs_retried
   }'
   ```

3. Check recent errors:
   ```bash
   docker compose logs norest-worker --tail 100 | grep ERROR
   ```