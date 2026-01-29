# PicoShare Performance Testing

This directory contains scripts for running performance tests on PicoShare using Firecracker microVMs.

## Quick Start

### 1. Initial Setup

Run the setup script to download and configure all necessary components:

```bash
./setup-test-environment
```

This will:
- Download Firecracker v1.7.0
- Download Linux kernel (vmlinux.bin)
- Download Ubuntu 22.04 base image
- Create an 8GB VM image with PicoShare installed
- Generate test files (100M, 500M, 1G, 2G, 5G)

**Note**: Setup requires sudo access for:
- Installing Firecracker to `/usr/local/bin`
- Adding your user to the `kvm` group
- Mounting/modifying the VM filesystem

### 2. Run Tests

Single test:
```bash
sudo ./run-test 2048 100M
```

Full test matrix:
```bash
./run-test-matrix
```

## Test Configuration

### Test Matrix

By default, tests run with the following configurations:

- **RAM limits**: 2048MB, 1024MB, 512MB, 256MB
- **File sizes**: 100M, 500M, 1G, 2G, 5G
- **Total tests**: 20 (4 RAM × 5 file sizes)

### Test Parameters

```bash
./run-test <RAM_MB> <FILE_SIZE>
```

Examples:
```bash
sudo ./run-test 2048 100M   # 2GB RAM, 100MB file
sudo ./run-test 1024 500M   # 1GB RAM, 500MB file
sudo ./run-test 512 1G      # 512MB RAM, 1GB file
```

## Results

Test results are saved to `results/` in JSON format:

```bash
# View latest result
cat results/result-fc-*.json | jq .

# List all results
ls -lt results/

# View test matrix summary
cat results/matrix-fc-*/summary.txt
```

### Result Format

Each test produces a JSON file with:

```json
{
  "timestamp": "2026-01-29T12:00:00+00:00",
  "platform": "firecracker",
  "ram_mb": 2048,
  "file_size": "100M",
  "file_size_mb": 100,
  "boot_time_seconds": 0.10,
  "duration_seconds": 0.80,
  "throughput_mbps": 125.00,
  "http_status": "200",
  "exit_reason": "success",
  "success": true
}
```

## Performance

### Firecracker Advantages

- **Boot time**: ~0.10 seconds (590x faster than traditional VMs)
- **Memory overhead**: ~5MB per VM (100x more efficient)
- **Test duration**: ~15 seconds per test (total ~5 minutes for full matrix)
- **Isolation**: Each test runs in a fresh microVM

### Typical Test Timing

- VM boot: 0.10s
- PicoShare startup: 10s
- File upload: varies by size (0.7s for 100M, 4s for 1G, 20s for 5G)
- Total per test: 11-35s depending on file size

## Troubleshooting

### Network Issues

If you see "No route to host" errors:

```bash
# Clean up stale routes
sudo ip route del 172.16.0.0/24 2>/dev/null || true

# Then retry the test
sudo ./run-test 2048 100M
```

### KVM Access

If you get "Permission denied" for `/dev/kvm`:

```bash
# Add your user to kvm group
sudo usermod -aG kvm $USER

# Log out and back in, then verify
groups | grep kvm
```

### Disk Space

The VM image is 8GB to support uploads up to 5GB. If tests fail with "no space left on device", ensure:

- Your host has enough space in `/home/mike/picoshare/perf-test/`
- The VM image hasn't been corrupted (re-run `./setup-test-environment`)

### Debug Mode

To see detailed VM console output:

```bash
# Console logs are written to /tmp/fc-console-*.log during test execution
tail -f /tmp/fc-console-*.log
```

## Architecture

### Components

```
firecracker-images/
├── vmlinux.bin              # Linux kernel (5.10)
├── ubuntu-22.04.ext4        # Base Ubuntu rootfs (300MB)
└── rootfs-working.ext4      # Modified VM image (8GB)
    ├── /usr/local/bin/picoshare
    └── /sbin/init-picoshare

test-files/
├── 100M.bin
├── 500M.bin
├── 1G.bin
├── 2G.bin
└── 5G.bin
```

### Network Configuration

- Host TAP device: `fc-tap-<PID>`
- Host IP: 172.16.0.1/24
- Guest IP: 172.16.0.2/24
- PicoShare port: 4001

### Custom Init

The VM uses a custom init script (`/sbin/init-picoshare`) that:
1. Mounts essential filesystems (proc, sys, dev)
2. Configures network interfaces
3. Starts PicoShare with test credentials
4. Keeps running as PID 1

## Advanced Usage

### Modify Test Matrix

Edit `run-test-matrix` to customize:

```bash
# Change RAM configurations
MEMORY_LIMITS=("2048" "1024" "512" "256")

# Change file sizes
FILE_SIZES=("100M" "500M" "1G" "2G" "5G")
```

### Rebuild VM Image

If you update PicoShare or need to modify the VM:

```bash
# Remove old image
rm firecracker-images/rootfs-working.ext4

# Re-run setup
./setup-test-environment
```

### Parallel Testing

While Firecracker supports running multiple VMs simultaneously, the current scripts run tests sequentially to avoid network conflicts. For parallel execution, you would need to:

1. Assign unique TAP devices and IPs per VM
2. Use different ports for each PicoShare instance
3. Coordinate cleanup across parallel test runners

## Requirements

- Linux x86_64 host
- KVM support (`/dev/kvm` accessible)
- 10GB+ free disk space
- Packages: `qemu-utils`, `curl`, `wget`, `jq`
- Sudo access for Firecracker and network setup

## Files

- `setup-test-environment` - Initial setup script
- `run-test` - Single test runner
- `run-test-matrix` - Full matrix test runner
- `FIRECRACKER-FINAL-SUMMARY.md` - Implementation details and results
