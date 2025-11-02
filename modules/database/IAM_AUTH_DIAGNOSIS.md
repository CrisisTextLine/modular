# IAM Authentication Token Rotation Diagnosis

## Problem Statement

Initial IAM authentication connection succeeds, but subsequent token rotation fails with PAM authentication errors.

## Root Cause Analysis

### How IAM Authentication Works

1. **Token Generation**: AWS RDS IAM tokens are valid for **15 minutes** from generation
2. **Library Used**: `github.com/davepgreene/go-db-credential-refresh`
3. **Expected Behavior**:
   - Library intercepts new database connections
   - Calls `Store.GetPassword()` to generate fresh IAM token
   - Each new connection gets a fresh token

### The Problem: Connection Lifetime vs Token Lifetime

IAM tokens expire after 15 minutes, but database connections can live longer. If:

```
ConnectionMaxLifetime >= 15 minutes
```

Then:
- Connection is created at T+0 with token (expires at T+15min)
- Connection stays alive past T+15min
- Next query on same connection uses **expired token**
- Result: **PAM authentication failure**

## Identified Failure Scenarios

### 1. Connection Outlives Token (Most Likely) ⚠️

**Symptom**: Works initially, fails after ~15 minutes

**Cause**: `ConnectionMaxLifetime` not set or >= 15 minutes

**Fix**: Set `connection_max_lifetime: "14m"` in configuration

### 2. Token Not Refreshed on New Connection

**Symptom**: New connections fail even with short lifetimes

**Cause**: `go-db-credential-refresh` library not calling `Store.GetPassword()`

**Debug**: Add logging to verify `GetPassword()` calls

### 3. Token Caching in awsrds.Store

**Symptom**: Consistent failures after first token expires

**Cause**: Store caches tokens beyond validity period

**Debug**: Inspect `awsrds.Store` implementation for TTL logic

### 4. Race Condition on Token Expiration

**Symptom**: Intermittent failures

**Cause**: Token expires between generation and connection

**Debug**: Log timestamps of token generation vs connection attempts

## Solution: Correct Configuration

### Critical Configuration Settings

```yaml
connections:
  default:
    driver: postgres
    dsn: "postgresql://user@host:5432/db?sslmode=require"

    # Connection pool settings
    max_open_connections: 10
    max_idle_connections: 2

    # ⚠️ CRITICAL: Must be LESS than 15 minutes
    connection_max_lifetime: "14m"    # Recycle connections before token expires
    connection_max_idle_time: "10m"   # Close idle connections early

    # IAM authentication
    aws_iam_auth:
      enabled: true
      region: "us-east-1"
      db_user: "your_iam_user"  # Optional, extracted from DSN if not set
      connection_timeout: "10s"
```

### Why These Settings Work

| Setting | Value | Reason |
|---------|-------|--------|
| `connection_max_lifetime` | `14m` | Forces connection recycling **before** 15-minute token expiration |
| `connection_max_idle_time` | `10m` | Closes idle connections early to avoid stale tokens |
| `connection_timeout` | `10s` | Allows time for token generation and network latency |

### Connection Lifecycle with Correct Settings

```
T+0:00   Initial connection created → Token valid until T+15:00
T+0:30   Query 1 succeeds (using connection from pool)
T+2:00   Query 2 succeeds (using connection from pool)
T+14:00  Connection reaches max lifetime → Closed by pool
T+14:05  Query 3 arrives → Pool creates NEW connection → Fresh token generated (valid until T+29:00)
T+14:10  Query 3 succeeds with fresh token ✓
```

### Without Correct Settings (Failure Mode)

```
T+0:00   Initial connection created → Token valid until T+15:00
T+0:30   Query 1 succeeds
T+15:05  Token EXPIRES
T+15:10  Query 2 arrives → Uses EXISTING connection → Token expired → ❌ PAM failure
```

## Testing and Verification

### Run Diagnostic Tests

```bash
cd modules/database
go test -v -run "TestIAMAuthDiagnosis"
go test -v -run "TestProblemSummary"
go test -v -run "TestConnectionLifetimeRecommendations"
```

### Run Integration Test (Requires AWS Credentials)

```bash
# Set environment variables
export TEST_IAM_RDS_ENDPOINT="your-db.region.rds.amazonaws.com:5432"
export TEST_IAM_RDS_REGION="us-east-1"
export TEST_IAM_DB_USER="your_iam_user"
export TEST_IAM_DB_NAME="your_database"

# Run test
go test -v -run "TestIAMTokenRotationScenario"
```

This test will:
- Create connections with aggressive recycling (10s lifetime)
- Query every 2 seconds for 45 seconds
- Monitor for PAM failures
- Report connection pool statistics

### Monitor Production Database

Use these SQL queries to verify connection behavior:

```sql
-- Check active connections and their age
SELECT pid,
       usename,
       application_name,
       client_addr,
       backend_start,
       state,
       age(now(), backend_start) as connection_age
FROM pg_stat_activity
WHERE usename = 'your_iam_username'
ORDER BY backend_start;
```

**Expected**: No connection should be older than 14 minutes

## Debugging Steps

### 1. Verify Current Configuration

Check your configuration file for `connection_max_lifetime`:

```bash
grep -A 10 "connection_max_lifetime" config.yaml
```

If missing or >= 15m, update it to `14m`.

### 2. Add Connection Pool Monitoring

```go
// Log connection stats periodically
ticker := time.NewTicker(1 * time.Minute)
go func() {
    for range ticker.C {
        stats := db.Stats()
        logger.Info("Connection pool stats",
            "open", stats.OpenConnections,
            "in_use", stats.InUse,
            "idle", stats.Idle,
            "max_lifetime_closed", stats.MaxLifetimeClosed,
        )
    }
}()
```

Watch for `MaxLifetimeClosed` increasing over time.

### 3. Add Token Generation Logging

To verify `Store.GetPassword()` is being called, you can:

1. Fork `go-db-credential-refresh` library
2. Add logging in the connector's `Connect()` method
3. Use your fork temporarily for debugging

Or use AWS CloudTrail to monitor `rds:GenerateDBAuthToken` API calls.

### 4. Test Token Expiration Manually

```go
// Generate a token manually
import "github.com/aws/aws-sdk-go-v2/feature/rds/auth"

authToken, err := auth.BuildAuthToken(
    ctx,
    endpoint,
    region,
    username,
    awsConfig.Credentials,
)

// Use token in connection string
dsn := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=require",
    username, authToken, endpoint, dbname)

// Try connection immediately - should work
db, err := sql.Open("postgres", dsn)

// Wait 15+ minutes
time.Sleep(16 * time.Minute)

// Try connection again - should fail with PAM error
err = db.Ping() // ❌ Expected to fail
```

## Common Error Messages

| Error Message | Meaning | Solution |
|---------------|---------|----------|
| `PAM authentication failed for user` | Token expired or invalid | Verify token is fresh; check ConnectionMaxLifetime |
| `password authentication failed` | IAM token rejected | Check IAM policy allows `rds-db:connect` |
| `no pg_hba.conf entry` | Database not configured for IAM | Check RDS parameter group |
| `context deadline exceeded` | Connection timeout | Increase `connection_timeout` |

## Verification Checklist

After implementing the fix:

- [ ] `connection_max_lifetime` set to `14m` (or less)
- [ ] `connection_max_idle_time` set to `10m` (or less)
- [ ] Diagnostic tests pass
- [ ] Monitor `db.Stats().MaxLifetimeClosed` increases over time
- [ ] Run application for > 15 minutes without PAM errors
- [ ] Check CloudTrail for periodic `GenerateDBAuthToken` calls
- [ ] Verify no connections older than 14 minutes in `pg_stat_activity`

## Summary

**The primary issue is likely that ConnectionMaxLifetime is not set or is >= 15 minutes, allowing connections to outlive their IAM tokens.**

**The fix is simple: Set `connection_max_lifetime: "14m"` in your database configuration.**

This ensures all connections are recycled before their IAM tokens expire, triggering fresh token generation for each new connection.

## Additional Resources

- [AWS RDS IAM Authentication Documentation](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html)
- [go-db-credential-refresh Library](https://github.com/davepgreene/go-db-credential-refresh)
- [PostgreSQL Connection Pooling Best Practices](https://www.postgresql.org/docs/current/runtime-config-connection.html)

## Files Created for Diagnosis

1. `iam_rotation_debug_test.go` - Integration test for token rotation with real AWS credentials
2. `iam_diagnosis_test.go` - Diagnostic tests explaining the issue (no AWS required)
3. `IAM_AUTH_DIAGNOSIS.md` - This document

Run the diagnostic tests anytime to understand the IAM authentication flow and troubleshoot issues.
