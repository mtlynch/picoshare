# PicoShare Performance Test Suite

Measures upload performance and stability under memory constraints.

## Prerequisites

- Vagrant
- VirtualBox or libvirt
- curl
- jq
- ~15GB free disk space for test files

## Usage

```bash
# Run full test matrix
./run-tests

# Compare two runs
./compare-runs results/<baseline> results/<compare>
```

## Test Matrix

| File Size | Memory Limit |
|-----------|--------------|
| 100MB, 500MB, 1GB, 2GB, 5GB | 256MB, 512MB, 1GB, 2GB |

20 total test cases.

## Output

Results are written to `results/<timestamp>/`:

- `raw.csv` - Machine-readable results
- `report.md` - Human-readable summary
- `failures.log` - Crash details and dmesg output
- `config.json` - Git commit, branch, test parameters

## Comparing Branches

```bash
git checkout main
./perf-test/run-tests
# Note: results/20250101-120000

git checkout feature-branch
./perf-test/run-tests
# Note: results/20250101-130000

./perf-test/compare-runs results/20250101-120000 results/20250101-130000
```
