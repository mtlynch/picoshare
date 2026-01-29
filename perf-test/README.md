# PicoShare Performance Testing

This directory contains tools for performance testing PicoShare with varying memory constraints and file sizes.

## Quick Start

### Run a Single Test

```bash
./run-test --ram 2048 --file-size 100M
```

### Run the Full Test Matrix

```bash
./run-test-matrix
```

This runs 20 tests (4 memory configs × 5 file sizes):
- **Memory**: 2048MB, 1024MB, 512MB, 256MB (descending)
- **File sizes**: 100M, 500M, 1G, 2G, 5G (ascending)

## Test Scripts

### `run-test`

Runs a single performance test with specified RAM and file size.

**Usage:**
```bash
./run-test --ram <MB> --file-size <SIZE> [OPTIONS]

Required:
  --ram MB          RAM limit in MB (e.g., 2048, 1024, 512, 256)
  --file-size SIZE  File size to test (e.g., 100M, 500M, 1G, 2G, 5G)

Optional:
  --timeout SEC     Upload timeout in seconds (default: 600)
  --keep-vm         Don't destroy VM after test
  --results-dir DIR Directory for results (default: ./results)
```

**Output:**
- Test results logged to stderr
- Result file path printed to stdout (JSON format)
- Exit code 0 on success, non-zero on failure

**Example:**
```bash
$ ./run-test --ram 512 --file-size 1G
[2026-01-29 12:00:00] === PicoShare Performance Test ===
[2026-01-29 12:00:00] RAM: 512MB
[2026-01-29 12:00:00] File: 1G
...
[2026-01-29 12:05:30] ✅ TEST PASSED
[2026-01-29 12:05:30]    Duration: 320.45s
[2026-01-29 12:05:30]    Throughput: 3.12 MB/s
[2026-01-29 12:05:30]    Peak Memory: 487MB / 512MB
[2026-01-29 12:05:30]    HTTP Status: 200
./results/result-512MB-1G-20260129-120000.json
```

### `run-test-matrix`

Orchestrates multiple `run-test` invocations for the full test matrix.

**Usage:**
```bash
./run-test-matrix [OPTIONS]

Options:
  --timeout SEC         Timeout per test (default: 600)
  --stop-on-failure     Stop entire matrix on first failure
  --memory-limits LIST  Override memory limits (comma-separated)
  --file-sizes LIST     Override file sizes (comma-separated)
```

**Output:**
- Individual test logs in results directory
- CSV summary: `results/matrix-TIMESTAMP/matrix-results.csv`
- Text summary: `results/matrix-TIMESTAMP/summary.txt`

**Example:**
```bash
$ ./run-test-matrix --memory-limits "2048,1024" --file-sizes "100M,500M"
[2026-01-29 12:00:00] PicoShare Performance Test Matrix
[2026-01-29 12:00:00] Memory configs: 2048 1024
[2026-01-29 12:00:00] File sizes: 100M 500M
[2026-01-29 12:00:00] Total tests: 4
...
[2026-01-29 12:15:30] Test matrix complete!
[2026-01-29 12:15:30]   Passed: 4
[2026-01-29 12:15:30]   Failed: 0
```

## Prerequisites

### System Requirements
- Vagrant ≥ 2.4.0
- libvirt provider for Vagrant
- KVM support
- At least 4GB available RAM
- At least 150GB free disk space (for test files and VMs)

### Installed Tools
- `curl` - for HTTP requests
- `jq` - for JSON parsing
- `bc` - for calculations
- `awk` - for text processing

### One-Time Setup

1. **Install dependencies:**
   ```bash
   # On Debian/Ubuntu:
   sudo apt-get install vagrant vagrant-libvirt qemu-kvm libvirt-daemon-system jq bc
   ```

2. **Generate test files:**
   ```bash
   ./generate-test-files
   ```
   This creates binary test files (100M, 500M, 1G, 2G, 5G) in `test-files/`.
   Total size: ~8.6GB

3. **Verify Vagrant setup:**
   ```bash
   vagrant status
   ```

## Test Results

Results are stored in `results/` directory (git-ignored).

### Single Test Result (JSON)

```json
{
  "timestamp": "2026-01-29T12:00:00+00:00",
  "ram_mb": 2048,
  "file_size": "1G",
  "file_size_mb": 1024,
  "duration_seconds": 89.23,
  "throughput_mbps": 11.48,
  "initial_memory_bytes": 157286400,
  "peak_memory_bytes": 524288000,
  "peak_memory_mb": 500,
  "http_status": "200",
  "exit_reason": "success",
  "success": true
}
```

### Matrix Results (CSV)

```csv
ram_mb,file_size,duration_seconds,throughput_mbps,peak_memory_mb,http_status,exit_reason,success,result_file
2048,100M,8.74,11.44,245,200,success,true,./results/result-2048MB-100M-...json
2048,500M,45.12,11.08,312,200,success,true,./results/result-2048MB-500M-...json
...
```

## Architecture

### Test Workflow

1. **VM Lifecycle**: Each test runs in an isolated Vagrant VM with specified RAM
2. **PicoShare Startup**: Binary is synced to VM and started in a screen session
3. **Upload Test**: File is uploaded via HTTP POST, measuring duration and memory
4. **Result Collection**: Peak memory, throughput, and status are recorded
5. **Cleanup**: VM is destroyed to ensure clean state for next test

### Why Screen?

PicoShare is started in a screen session (`screen -dmS picoshare ...`) because:
- SSH backgrounding (`vagrant ssh -c "command &" -- -f`) leaves SSH session hung
- Screen properly daemonizes the process
- Allows inspecting PicoShare if test fails (`vagrant ssh` then `screen -r picoshare`)

See `FLAKINESS.md` for detailed analysis of testing challenges.

## Troubleshooting

### VM Won't Start
```bash
# Check Vagrant status
vagrant status

# View detailed VM logs
vagrant up

# Destroy and retry
vagrant destroy -f
```

### PicoShare Won't Start
```bash
# SSH into VM
vagrant ssh

# Check if PicoShare is running
ps aux | grep picoshare

# View PicoShare logs
cat /tmp/picoshare.log

# List screen sessions
screen -ls

# Attach to PicoShare screen
screen -r picoshare
```

### Test Hangs or Times Out
- Default timeout is 600 seconds (10 minutes) per test
- Increase with `--timeout` flag
- Check if VM has enough memory
- Monitor VM: `vagrant ssh` then `top` or `free -h`

### Vagrant Lock Error
```
An action 'up' was attempted on the machine 'default',
but another process is already executing an action...
```

**Solution:**
```bash
# Find vagrant processes
ps aux | grep vagrant

# Kill them
pkill -9 vagrant

# Clean up lock files
rm -rf .vagrant/machines/default/*/action_*
```

## Known Issues

See `FLAKINESS.md` for comprehensive documentation of:
- SSH backgrounding problems
- Rsync performance with large files
- Test retry strategies
- Recommendations for PicoShare improvements

## File Structure

```
perf-test/
├── README.md              # This file
├── FLAKINESS.md           # Testing challenges and solutions
├── Vagrantfile            # VM configuration
├── run-test               # Single test script
├── run-test-matrix        # Matrix orchestrator
├── generate-test-files    # Create test files
├── test-files/            # Binary test files (git-ignored)
│   ├── 100M.bin
│   ├── 500M.bin
│   ├── 1G.bin
│   ├── 2G.bin
│   └── 5G.bin
└── results/               # Test results (git-ignored)
    └── matrix-TIMESTAMP/
        ├── result-*.json
        ├── test-*.log
        ├── matrix-results.csv
        └── summary.txt
```

## Development

### Testing the Test Scripts

Run a quick single test:
```bash
./run-test --ram 2048 --file-size 100M --keep-vm
```

The `--keep-vm` flag preserves the VM for debugging.

### Adding New Test Configurations

Edit the arrays in `run-test-matrix`:
```bash
FILE_SIZES=("100M" "500M" "1G" "2G" "5G")
MEMORY_LIMITS=("2048" "1024" "512" "256")
```

### Debugging

Enable verbose Vagrant output:
```bash
VAGRANT_LOG=debug ./run-test --ram 2048 --file-size 100M
```

Enable bash tracing:
```bash
bash -x ./run-test --ram 2048 --file-size 100M
```

## Contributing

When adding new test scripts or modifying existing ones:
1. Follow the error handling pattern (`set -euo pipefail`)
2. Use the `log()` function for all user-facing messages
3. Use the `die()` function for fatal errors
4. Always clean up resources in trap handlers
5. Document new flags in usage() function
6. Add examples to this README

## License

Same as PicoShare main project.
