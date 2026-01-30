# Performance Testing Handoff

This document explains how to run comprehensive performance tests across the baseline and all optimization branches.

## Overview

We're comparing the baseline (`perf-test` branch) against 6 performance optimization branches:
1. `perf/batch-chunk-reading` - Fetch multiple chunks per query
2. `perf/connection-pool-tuning` - SQLite connection pool optimization
3. `perf/denormalize-file-size` - Store file size in entries table
4. `perf/downloads-index` - Add index on downloads.entry_id
5. `perf/larger-chunk-size` - Increase chunk size from 320KB to 4MB
6. `perf/ncruces2-combined` - Pure-Go SQLite driver (ncruces vs mattn)

## Test Configuration

### Test Matrix
- **RAM configurations**: 2048MB, 1024MB, 512MB, 256MB
- **File sizes**: 100M, 500M, 1G, 2G, 5G
- **Total tests per branch**: 20 (4 RAM × 5 file sizes)
- **Iterations per test**: 10 uploads with statistical analysis
- **Total uploads per branch**: 200 (20 tests × 10 iterations)

### System Requirements
- KVM virtualization support (`sg kvm -c` to run commands)
- Sudo access for Firecracker operations
- ~100GB disk space for VM images and test files
- ~4 hours per full branch test (all 20 tests)

## Prerequisites

### 1. Ensure test files exist
```bash
cd perf-test
ls -lh test-files/
# Should see: 100M.bin, 500M.bin, 1G.bin, 2G.bin, 5G.bin
```

If test files don't exist, create them:
```bash
cd perf-test
mkdir -p test-files
dd if=/dev/urandom of=test-files/100M.bin bs=1M count=100
dd if=/dev/urandom of=test-files/500M.bin bs=1M count=500
dd if=/dev/urandom of=test-files/1G.bin bs=1M count=1024
dd if=/dev/urandom of=test-files/2G.bin bs=1M count=2048
dd if=/dev/urandom of=test-files/5G.bin bs=1M count=5120
```

### 2. Set up Firecracker VM environment
```bash
cd perf-test
export PS_VERSION=dev
./setup-test-environment
```

This will:
- Download Firecracker and kernel
- Create a 50GB VM rootfs image
- Install PicoShare in the VM
- Configure networking

## Running Tests

### Option 1: Full Test Suite (All Branches)

Run tests for baseline + all 6 optimization branches sequentially:

```bash
cd perf-test

# Test each branch
for branch in "perf-test" "perf/batch-chunk-reading" "perf/connection-pool-tuning" \
              "perf/denormalize-file-size" "perf/downloads-index" \
              "perf/larger-chunk-size" "perf/ncruces2-combined"; do

    echo "=========================================="
    echo "Testing branch: $branch"
    echo "=========================================="

    # Checkout branch
    cd ..
    git checkout "$branch"
    cd perf-test

    # Rebuild VM environment (ensures correct binary and 50GB filesystem)
    export PS_VERSION=dev
    ./setup-test-environment

    # Run full test matrix (20 tests × 10 iterations = 200 uploads)
    sg kvm -c "./run-test-matrix"

    echo "✅ Completed: $branch"
    echo ""
done
```

**Estimated time**: 24-28 hours total (7 branches × ~4 hours each)

### Option 2: Test Single Branch

Test just one branch (faster for validation):

```bash
cd /home/mike/picoshare
git checkout perf-test  # or any perf/* branch
cd perf-test

# Rebuild VM
export PS_VERSION=dev
./setup-test-environment

# Run full matrix
sg kvm -c "./run-test-matrix"
```

**Estimated time**: 3-4 hours per branch

### Option 3: Quick Validation Test

Test a single configuration (useful for verifying fixes):

```bash
cd perf-test
sg kvm -c "./run-test 2048 5G"
```

**Estimated time**: ~20-30 minutes (10 iterations of 5GB upload)

## Test Output

### Results Directory Structure
```
perf-test/results/
├── matrix-perf-test-YYYYMMDD-HHMMSS/
│   ├── matrix-results.csv           # Aggregate CSV
│   ├── summary.txt                  # Human-readable summary
│   ├── test-2048MB-100M.log        # Individual test logs
│   ├── test-2048MB-500M.log
│   └── ...
├── result-fc-2048MB-100M-YYYYMMDD-HHMMSS.json  # Individual JSON results
└── ...
```

### Understanding Results

Each test produces:
- **JSON file**: Detailed per-iteration data with statistics
- **CSV entry**: Single row with median throughput
- **Log file**: Full test execution log

Key metrics in JSON:
```json
{
  "stats": {
    "median_throughput_mbps": 151.52,
    "avg_throughput_mbps": 149.05,
    "stddev_throughput_mbps": 14.24,
    "success_count": 10,
    "failure_count": 0
  }
}
```

## Comparing Results

### Using the compare-runs script

Compare two test runs:
```bash
cd perf-test
./compare-runs results/matrix-perf-test-TIMESTAMP1 results/matrix-perf-larger-chunk-size-TIMESTAMP2
```

Output shows throughput differences per configuration.

### Manual CSV Analysis

Extract median throughput for all 2048MB RAM tests:
```bash
# For each branch result directory
grep "^2048," results/matrix-*/matrix-results.csv
```

Create comparison table:
```bash
echo "Branch,100M,500M,1G,2G,5G" > comparison.csv
for branch in perf-test perf-batch-chunk-reading ...; do
    latest=$(find results -name "matrix-${branch}-*" -type d | sort | tail -1)
    echo -n "$branch,"
    grep "^2048," "$latest/matrix-results.csv" | awk -F',' '{print $5}' | tr '\n' ',' | sed 's/,$//'
    echo ""
done >> comparison.csv
```

## Known Issues & Solutions

### Issue: "no space left on device" on 5GB tests
**Solution**: VM filesystem is 50GB. If this occurs, check that you've run `setup-test-environment` after merging latest fixes.

```bash
# Verify VM size
qemu-img info perf-test/firecracker-images/rootfs-working.ext4 | grep "virtual size"
# Should show: virtual size: 50 GiB
```

### Issue: Tests hang or timeout
**Solution**: Database clear endpoint may have failed. Check logs:
```bash
tail -50 perf-test/results/test-*.log | grep -E "(clear|WARNING|ERROR)"
```

### Issue: Inconsistent results (high stddev)
**Possible causes**:
- System load during tests (ensure no other heavy processes)
- Disk I/O contention
- VM not properly reset between tests

**Solution**: Verify database clear is working:
```bash
# Should see "Database clear successful" between iterations
grep "clear" perf-test/results/test-*.log
```

## Test Infrastructure Details

### Key Files
- `run-test-matrix`: Orchestrates all 20 tests for a branch
- `run-test`: Runs a single test with 10 iterations
- `setup-test-environment`: Builds VM image and downloads dependencies
- `fc-network-config`: Network configuration for Firecracker
- `compare-runs`: Compares results between two test runs

### How Tests Work
1. **Build**: Compile PicoShare binary using `./dev-scripts/build-backend`
2. **VM Update**: Mount rootfs, copy binary, clean /tmp
3. **VM Boot**: Start Firecracker microVM with specified RAM
4. **Authenticate**: Log in to PicoShare admin
5. **Upload Loop**: Upload test file 10 times
   - Clear database via API between iterations
   - Measure upload duration and calculate throughput
6. **Statistics**: Calculate median, average, stddev
7. **Cleanup**: Stop VM, tear down network

### Database Clear Between Iterations
Each iteration (except the first) clears the database via HTTP API:
```bash
curl -X POST http://172.16.0.2:4001/api/debug/db/clear
```

This endpoint:
- Deletes all data from database tables
- Cleans up multipart temp files in /tmp
- Does NOT run VACUUM (too slow)

## Expected Performance Baseline

### Reference Results (perf-test branch, 2048MB RAM)
Based on successful 10-iteration tests:
- **100M**: ~150 MB/s median
- **500M**: ~130 MB/s median
- **1G**: ~85 MB/s median
- **2G**: ~70 MB/s median
- **5G**: ~35 MB/s median

Lower memory configurations will have reduced throughput.

## Performance Improvement Goals

### Optimizations to Test
1. **larger-chunk-size**: Expected +30-60% on large files
2. **denormalize-file-size**: Expected +10-20% on metadata queries
3. **connection-pool-tuning**: Expected +5-10% consistency improvement
4. **downloads-index**: Expected +20-30% on download-heavy workflows
5. **batch-chunk-reading**: Expected +10-15% on downloads
6. **ncruces2-combined**: Expected +20-40% with pure-Go driver

### Success Criteria
- All 20 tests pass with 10/10 iterations successful
- Median throughput improvement vs baseline
- Standard deviation remains low (<20% of mean)
- No regressions on any file size

## Next Steps

1. **Run full test suite** on all branches (24-28 hours)
2. **Collect results** into `perf-test/results/` directory
3. **Analyze comparisons** using compare-runs or CSV analysis
4. **Document findings** in a performance analysis report
5. **Identify best optimizations** for merging to master

## Questions or Issues

If tests fail or produce unexpected results:
1. Check logs in `perf-test/results/test-*.log`
2. Verify VM filesystem size (should be 50GB)
3. Confirm database clear API is working (check for "clear successful" in logs)
4. Ensure KVM permissions (`groups | grep kvm`)
5. Check disk space (`df -h`)

For debugging:
```bash
# Check VM can boot
cd perf-test
sg kvm -c "./run-test 256 100M"

# Verify database clear endpoint
curl -X POST http://172.16.0.2:4001/api/debug/db/clear
```
