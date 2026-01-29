# PicoShare Performance Test Flakiness Issues

## Summary
Automated performance testing revealed several issues that make testing fragile. This document describes the problems and suggests solutions.

## Issues Identified

### 1. SSH Backgrounding Problem (Critical)

**Problem:** The command `vagrant ssh -c "command &" -- -f` does not properly detach the SSH session. The SSH process remains open waiting for the backgrounded process, causing the test script to hang indefinitely.

**Evidence:**
- Multiple hung SSH processes observed: `ssh ... -f -t bash -l -c 'cd /vagrant && ... ./picoshare ... &'`
- These processes remain open for hours, blocking script progress
- The bash wrapper process in the VM also stays alive

**Root Cause:** SSH with `-f` flag goes to background *before* command execution, but when the command itself contains `&`, the SSH session still waits for all processes in the session to exit. The bash wrapper stays alive.

**Workarounds Attempted:**
- Adding `< /dev/null` to redirect stdin - didn't help
- Using `nohup` - didn't fully resolve the issue
- Using `-- -f` flag - still hangs

**Proper Solution:**
Instead of backgrounding through SSH, use a different approach:

```bash
# Option 1: Use systemd or supervisor in the VM
vagrant ssh -c "systemd-run --unit=picoshare --working-directory=/vagrant ./picoshare ..."

# Option 2: Write a startup script in the VM that daemonizes properly
cat > /tmp/start-picoshare.sh <<'EOF'
#!/bin/bash
cd /vagrant
nohup ./picoshare "$@" </dev/null >/dev/null 2>&1 &
disown
EOF
vagrant ssh < /tmp/start-picoshare.sh

# Option 3: Use screen or tmux in the VM
vagrant ssh -c "screen -dmS picoshare bash -c 'cd /vagrant && ./picoshare ...'"

# Option 4 (current workaround): Don't background, use separate SSH connections
```

### 2. Rsync Performance with Large Files

**Problem:** Syncing 8.6GB of test files takes ~2 minutes on each VM restart, adding significant overhead to tests.

**Impact:**
- Each memory configuration requires VM restart
- 2+ minutes of rsync time × 4 memory configs = 8+ minutes of overhead
- Total test time dominated by rsync, not actual testing

**Solutions:**
1. **Use NFS instead of rsync** (requires additional setup but faster for large files)
2. **Generate test files inside the VM** (trades upload time for generation time)
3. **Use shared storage** (requires infrastructure changes)
4. **Cache test files** in a base box snapshot (best option)

### 3. Lack of Structured Logging

**Problem:** The monolithic `run-tests` script mixes VM management, file uploads, and result collection. When something fails, it's hard to determine what went wrong.

**Solution:**
- Break into smaller, testable units
- Single test script: `run-test --ram 2048 --file-size 100M`
- Matrix runner: `run-test-matrix` orchestrates single tests
- Clear separation of concerns: VM management, PicoShare lifecycle, test execution

### 4. No Test Retry Logic

**Problem:** Transient failures (network hiccups, VM boot delays) cause entire test suite to abort.

**Solution:** Add retry logic with exponential backoff for:
- VM boot/SSH connectivity
- PicoShare startup verification
- File upload attempts (with new PicoShare instance)

### 5. Error Handling with `set -euo pipefail`

**Problem:** While strict error handling is good, it causes the entire test suite to abort on any single failure. We want to collect results for successful tests even if some fail.

**Solution:**
- Use `set -euo pipefail` within individual test runs
- Catch failures at the matrix level and continue with remaining tests
- Collect partial results

## Recommendations for PicoShare

### 1. Add Health Check Endpoint

**Current:** Must poll the UI endpoint to check readiness
**Suggested:** Add `/api/health` endpoint that returns immediately

```json
GET /api/health
{
  "status": "ok",
  "version": "1.0.0",
  "uptime_seconds": 42
}
```

### 2. Add Graceful Shutdown Signal Handler

**Current:** PicoShare is killed with SIGTERM/SIGKILL during test cleanup
**Suggested:** Handle SIGTERM gracefully to:
- Complete in-flight uploads
- Close database cleanly
- Prevent database corruption

### 3. Add Metrics Endpoint

**Current:** Must poll VM memory externally
**Suggested:** Expose internal metrics

```json
GET /api/metrics
{
  "memory_usage_bytes": 104857600,
  "goroutines": 42,
  "uploads_active": 1,
  "uploads_total": 157
}
```

**Note:** Since PicoShare uses CGO (for SQLite), Go's standard runtime metrics may not be accurate for memory tracking. Consider using OS-level metrics or implementing custom memory tracking that accounts for CGO allocations.

### 4. Add Upload Progress API

**Current:** Client has no visibility into upload progress
**Suggested:** Server-sent events or WebSocket for upload progress

Benefits:
- Better test diagnostics
- Could detect if upload stalls
- Better user experience

### 5. Configuration Validation

**Current:** Silent failures if environment variables are wrong
**Suggested:** Validate configuration on startup and fail fast with clear messages

## Test Strategy Improvements

### Current Structure (Flaky)
```
run-tests (monolithic)
├── Build PicoShare
├── Generate test files
└── For each memory config:
    ├── Start VM (hangs here due to SSH issue)
    ├── Start PicoShare
    ├── For each file size:
    │   └── Upload test
    └── Stop VM
```

### New Structure (Robust)
```
run-test-matrix
└── For each memory config:
    └── For each file size:
        └── run-test --ram 2048 --file-size 100M
            ├── Start VM (if needed)
            ├── Start PicoShare (proper daemonization)
            ├── Wait for ready (with timeout)
            ├── Run upload test
            ├── Collect results
            └── Cleanup (trap on exit)
```

Benefits:
- Each test is independent and can be run manually
- Failures in one test don't abort the suite
- Easy to debug single test case
- Can parallelize tests across multiple VMs
- Clear separation of concerns

## Next Steps

1. ✅ Document flakiness issues (this file)
2. ⏳ Implement `run-test` script for single tests
3. ⏳ Implement `run-test-matrix` orchestrator
4. ⏳ Add retry logic and better error handling
5. ⏳ Consider PicoShare improvements (health endpoint, metrics)
6. ⏳ Evaluate test file generation in VM vs rsync
