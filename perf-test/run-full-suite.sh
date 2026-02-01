#!/bin/bash
set -e

# Test each branch
for branch in "perf-test" "perf/batch-chunk-reading" "perf/connection-pool-tuning" \
              "perf/denormalize-file-size" "perf/downloads-index" \
              "perf/larger-chunk-size" "perf/ncruces2-combined"; do

    echo "=========================================="
    echo "Testing branch: $branch"
    echo "Started: $(date)"
    echo "=========================================="

    # Checkout branch
    cd /home/mike/picoshare
    git checkout "$branch"
    cd perf-test

    # Rebuild VM environment (ensures correct binary and 50GB filesystem)
    export PS_VERSION=dev
    ./setup-test-environment

    # Run full test matrix (20 tests × 10 iterations = 200 uploads)
    sg kvm -c "./run-test-matrix"

    echo "✅ Completed: $branch at $(date)"
    echo ""
done

echo "=========================================="
echo "🎉 ALL TESTS COMPLETE"
echo "Finished: $(date)"
echo "=========================================="
