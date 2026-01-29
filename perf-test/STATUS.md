# Performance Test Suite Status

## Summary

The performance test infrastructure has been restructured to be more robust and debuggable. Manual tests work reliably, but the automated test scripts need final verification and debugging.

## What Was Accomplished

### 1. Root Cause Analysis ✅

Identified the core issue preventing automated tests from running:

**Problem:** `vagrant ssh -c "command &" -- -f` does not properly detach SSH sessions, leaving them hung indefinitely.

**Evidence:**
- Multiple SSH processes found stuck for hours
- Bash wrapper processes remain alive in VM
- Test scripts hang waiting for SSH to complete

**Solution:** Use `screen -dmS picoshare` to properly daemonize PicoShare in the VM.

### 2. Test Infrastructure Restructure ✅

Created a new, modular test architecture:

**Before (Monolithic):**
- Single `run-tests` script handled everything
- Hard to debug failures
- All-or-nothing execution

**After (Modular):**
- `run-test`: Single test execution (can be run manually)
- `run-test-matrix`: Orchestrates multiple tests
- Clear separation of concerns
- Individual tests can be debugged

### 3. Documentation ✅

Created comprehensive documentation:

- **README.md**: User guide with examples, troubleshooting, architecture
- **FLAKINESS.md**: Detailed analysis of testing challenges and solutions
- **STATUS.md**: This file - current status and next steps

### 4. Configuration Updates ✅

- Updated Vagrantfile to install `screen` during provisioning
- Updated .gitignore to exclude test artifacts and logs
- Added note about CGO and Go metrics inaccuracy

## Manual Test Results ✅

Manual tests confirmed the infrastructure works:

| File Size | RAM   | Duration | Throughput | Status |
|-----------|-------|----------|------------|--------|
| 100M      | 2048MB| 17.3s    | ~5.8 MB/s  | ✅ PASS |
| 500M      | 2048MB| 100.1s   | ~5.0 MB/s  | ✅ PASS |
| 1G        | 2048MB| 192.2s   | ~5.3 MB/s  | ✅ PASS |
| 2G        | 2048MB| 263.2s   | ~7.8 MB/s  | ✅ PASS |
| 5G        | 2048MB| (interrupted)| -      | ⏸️ IN PROGRESS |

All manual uploads completed successfully with HTTP 200.

## Current Status

### Working ✅
- VM provisioning with Vagrant + libvirt
- PicoShare builds and runs in VM
- Authentication via API
- File uploads (tested up to 2GB)
- Screen-based daemonization
- Memory monitoring
- Result JSON generation

### Needs Verification ⚠️
- `run-test` script end-to-end execution
- Proper error handling and cleanup
- Memory monitoring during upload
- Result file generation
- `run-test-matrix` orchestration

### Known Issues ⚠️

1. **Test script cleanup timing**: Script exits quickly after starting PicoShare readiness check (needs debugging)
2. **Vagrant lock conflicts**: Multiple test runs can conflict (need better locking)
3. **Rsync overhead**: ~2 minutes to sync 8.6GB per VM restart

## Recommendations for PicoShare

Documented in `FLAKINESS.md`:

1. **Add `/api/health` endpoint** for faster readiness checks
2. **Handle SIGTERM gracefully** for clean shutdown
3. **Add `/api/metrics` endpoint** for memory/upload metrics
   - Note: CGO usage means Go runtime metrics won't be accurate
4. **Add upload progress API** for better observability
5. **Validate configuration on startup** with clear error messages

## Next Steps

### Immediate (Required for automated tests)

1. **Debug `run-test` script**:
   - Find why it exits early after "Waiting for PicoShare to be ready"
   - Verify readiness check loop executes properly
   - Test complete flow: VM start → PicoShare start → upload → cleanup

2. **Verify test with --keep-vm**:
   ```bash
   ./run-test --ram 2048 --file-size 100M --keep-vm
   ```
   This preserves the VM for debugging.

3. **Test single run end-to-end**:
   ```bash
   ./run-test --ram 2048 --file-size 100M
   ```
   Verify result JSON is created and accurate.

4. **Test matrix runner with subset**:
   ```bash
   ./run-test-matrix --memory-limits "2048" --file-sizes "100M,500M"
   ```
   Test orchestration with just 2 tests.

### Short-term (Robustness)

1. **Add retry logic** for transient failures
2. **Better error messages** with context
3. **Vagrant lock handling** - detect and clear stale locks
4. **Partial result preservation** - save results even if later tests fail

### Long-term (Optimization)

1. **Pre-baked VM image** with test files already synced
2. **Parallel test execution** (requires multiple VMs)
3. **Test file generation in VM** to avoid rsync overhead
4. **Continuous test running** with result database

## Files Created/Modified

### New Files
- `perf-test/run-test` - Single test script
- `perf-test/run-test-matrix` - Matrix orchestrator
- `perf-test/FLAKINESS.md` - Testing challenges documentation
- `perf-test/README.md` - User guide
- `perf-test/STATUS.md` - This file

### Modified Files
- `perf-test/Vagrantfile` - Added screen installation
- `.gitignore` - Added test log exclusions

## Test Matrix When Ready

When automated tests are working, the full matrix will run:

**20 tests total:**
- RAM: 2048MB, 1024MB, 512MB, 256MB (4 configs)
- Files: 100M, 500M, 1G, 2G, 5G (5 sizes)

**Estimated duration:**
- VM startup: ~4 min/config × 4 = ~16 min
- Uploads: Variable (faster with more RAM)
- Total: ~2-4 hours for full matrix

## How to Resume

To continue testing:

1. **Clean up current state:**
   ```bash
   cd /home/mike/picoshare/perf-test
   vagrant destroy -f
   pkill -9 vagrant || true
   ```

2. **Test single test script:**
   ```bash
   ./run-test --ram 2048 --file-size 100M --keep-vm 2>&1 | tee test-debug.log
   ```

3. **If successful, test matrix with subset:**
   ```bash
   ./run-test-matrix --memory-limits "2048,1024" --file-sizes "100M"
   ```

4. **When confident, run full matrix:**
   ```bash
   ./run-test-matrix
   ```

## Questions for Review

1. Should we add test retry logic, or keep tests deterministic?
2. Is 600s timeout per test sufficient for 5GB uploads on 256MB RAM?
3. Should we generate test files in VM to avoid rsync overhead?
4. Do we want parallel test execution (requires infrastructure changes)?

## References

- **Manual test evidence**: Earlier in this conversation (100M, 500M, 1G, 2G all passed)
- **SSH backgrounding issue**: Process tree shows hung SSH sessions
- **FLAKINESS.md**: Comprehensive analysis with solutions
- **README.md**: Complete user guide
