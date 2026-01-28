# PicoShare Performance Test Suite

Measures upload performance and stability under memory constraints.

## Prerequisites

- Vagrant
- VirtualBox or libvirt
- curl
- jq
- Go 1.24+ (provided by nix flake)
- ~15GB free disk space for test files

### Dependency Management

This project uses Nix for reproducible builds. Use `nix develop` to enter a shell with all dependencies (Go 1.24, Node.js, SQLite, etc.) at pinned versions.

### libvirt Setup

When using libvirt provider:
1. Add user to libvirt group: `sudo usermod -aG libvirt $USER`
2. Log out and back in (or use `sg libvirt -c 'command'`) for group changes to take effect
3. Ensure libvirtd is running: `sudo systemctl start libvirtd`

**Known Issue:** First-time VM boot with vagrant-libvirt may take 10-15 minutes to acquire an IP address. VMs with <512MB RAM may struggle to boot Debian 12 in reasonable time.

### Troubleshooting Networking

If VM gets stuck on "Waiting for domain to get an IP address...":

1. **Ensure default network is active:**
   ```bash
   sudo virsh net-start default
   sudo virsh net-autostart default
   sudo virsh net-list --all
   ```

2. **Check DHCP leases:**
   ```bash
   for net in $(virsh net-list --name); do virsh net-dhcp-leases ${net}; done
   ```

3. **Reset vagrant-libvirt network (if exists):**
   ```bash
   vagrant destroy --force
   sudo virsh net-destroy vagrant-libvirt
   sudo virsh net-undefine vagrant-libvirt
   vagrant up
   ```

4. **Check VM console:**
   ```bash
   sudo virsh screenshot perf-test_default /tmp/vm-screenshot.ppm
   sudo virsh domifaddr perf-test_default
   ```

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
| 5GB, 2GB, 1GB, 500MB, 100MB | 2GB, 1GB, 512MB, 256MB |

20 total test cases. Tests run from largest resource VM to smallest to optimize boot times and identify issues early.

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
