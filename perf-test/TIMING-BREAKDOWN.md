# Performance Test Timing Breakdown

## Single Test Analysis (100M file, 2048MB RAM)

Based on actual test run from 2026-01-29 00:54:58 to 00:56:27 (total: **1m 29s**)

### Phase-by-Phase Breakdown

| Phase | Duration | % of Total | Cumulative | Details |
|-------|----------|------------|------------|---------|
| **1. Lock Check** | 2s | 2% | 2s | Detect vagrant lock conflicts |
| **2. VM Startup** | 59s | 66% | 61s | `vagrant up` - boot Ubuntu VM |
| **3. Copy Binary** | 4s | 4% | 65s | SCP PicoShare binary (~15MB) |
| **4. Start PicoShare** | 5s | 6% | 70s | Launch via screen, wait for process |
| **5. Readiness Check** | 2s | 2% | 72s | Wait for HTTP endpoint to respond |
| **6. Process Verification** | 1s | 1% | 73s | Verify PicoShare still running |
| **7. Authentication** | 2s | 2% | 75s | POST to /api/auth, get cookie |
| **8. File Upload** | 10s | 11% | 85s | Upload 100M file via HTTP |
| **9. Result Processing** | 1s | 1% | 86s | Calculate metrics, write JSON |
| **10. Cleanup** | 3s | 3% | 89s | Destroy VM |

### Key Insights

**Bottlenecks:**
1. 🐌 **VM Startup (59s, 66%)** - Largest time sink
   - Boot Ubuntu kernel
   - Initialize systemd services
   - Configure networking (DHCP)
   - SSH key exchange
   - Run provisioning script (apt-get install screen)

2. 📁 **File Upload (10s, 11%)** - The actual test
   - 100MB in 9.63s = 10.38 MB/s throughput
   - This is what we're measuring!

3. 🔧 **Setup Overhead (15s, 17%)** - Everything else
   - Copy binary: 4s
   - Start PicoShare: 5s
   - Checks and auth: 6s

### Comparison with Old Method (8.6GB rsync)

| Method | VM Startup | Setup | Upload | Total |
|--------|------------|-------|--------|-------|
| **Old (rsync 8.6GB)** | 180-240s | 15s | 10s | ~4-5 min |
| **New (SCP binary)** | 59s | 15s | 10s | **~1.5 min** |
| **Improvement** | ⬇️ 67% | - | - | **⬇️ 70%** |

### Scaling to Different File Sizes

Based on 100M baseline, estimated timing for full test matrix:

| File Size | Upload Time | Total Test Time | Notes |
|-----------|-------------|-----------------|-------|
| **100M** | ~10s | ~1.5 min | ✅ Measured |
| **500M** | ~50s | ~2 min | 5x file size |
| **1G** | ~100s | ~3 min | Network sustained |
| **2G** | ~200s | ~4.5 min | May OOM on low RAM |
| **5G** | ~500s | ~9 min | May OOM on low RAM |

### Full Matrix Estimation (20 tests)

**Optimistic scenario (all tests pass):**
```
VM overhead: 20 × 74s (setup) = 24.7 minutes
Uploads:
  - 4 × 100M = 4 × 10s = 40s
  - 4 × 500M = 4 × 50s = 200s
  - 4 × 1G = 4 × 100s = 400s
  - 4 × 2G = 4 × 200s = 800s
  - 4 × 5G = 4 × 500s = 2000s

Total uploads: 3440s = 57 minutes
Total time: 24.7 + 57 = ~82 minutes (~1.4 hours)
```

**Realistic scenario (some OOM failures, retries):**
- OOM failures on 256MB/512MB RAM with 2G/5G files
- Add 20% overhead for failures and retries
- **Estimated: ~1.7-2 hours**

### Where Can We Optimize Further?

#### Option 1: Pre-baked VM Image ⚡⚡⚡
**Impact:** VM startup 59s → ~10s (saves 49s × 20 = 16 minutes)

Create a Vagrant box with:
- PicoShare binary already installed
- Screen already installed
- No provisioning needed

```bash
# One-time: create custom box
vagrant up
vagrant ssh -c "sudo apt-get install -y screen"
vagrant scp picoshare :~/picoshare
vagrant package --output picoshare-test.box

# Per test: use custom box
config.vm.box = "picoshare-test"  # Boots in ~10s
```

#### Option 2: Keep VM Running Between Tests ⚡⚡
**Impact:** VM startup 59s → 0s on subsequent tests

Instead of destroy/create, just restart PicoShare:
```bash
# First test: create VM (59s)
# Tests 2-20: reuse VM (0s startup)
# Last test: destroy VM

Savings: 59s × 19 tests = 18.7 minutes
```

**Tradeoff:** Less isolation, potential state contamination between tests

#### Option 3: Parallel Testing ⚡⚡⚡
**Impact:** 82 minutes → ~20 minutes (4x speedup)

Run 4 VMs in parallel (4 different memory configs):
```bash
# In parallel:
./run-test --ram 2048 --file-size 100M &
./run-test --ram 1024 --file-size 100M &
./run-test --ram 512 --file-size 100M &
./run-test --ram 256 --file-size 100M &
wait
```

**Requires:** Unique port forwarding per VM, unique VM names

#### Option 4: Switch to Firecracker ⚡⚡⚡⚡
**Impact:** VM startup 59s → ~0.2s (saves 59s × 20 = 20 minutes)

MicroVM boots in milliseconds:
```
Total test time per run: ~15s (setup + upload)
Full matrix: 20 × 15s = 5 minutes + uploads (57 min) = ~62 minutes
```

### Recommended Optimizations (Prioritized)

1. **Quick wins (do now):**
   - ✅ Disable synced folder (DONE - saved 2-3 min/test)
   - [ ] Pre-bake VM image with PicoShare + screen (saves 16 min total)

2. **Medium effort (do if running tests regularly):**
   - [ ] Keep VM running between same-RAM tests (saves 19 min)
   - [ ] Parallel testing (saves 60 min, needs 16GB host RAM)

3. **Long-term (if tests become critical):**
   - [ ] Switch to Firecracker (saves 20 min startup + enables parallelism)

### Current Status

With the synced folder fix, we're at:
- **Single test:** ~1.5 minutes
- **Full matrix:** ~1.7-2 hours (estimated)

This is **reasonable** for occasional performance testing. Further optimization only needed if running tests frequently (daily/weekly).

## Actual Test Performance Data

From successful 100M upload tests:

| Test | RAM | Upload Time | Throughput | Peak Memory |
|------|-----|-------------|------------|-------------|
| 1 | 2048MB | 14.18s | 7.05 MB/s | 341MB |
| 2 | 2048MB | 14.38s | 6.95 MB/s | 343MB |
| 3 | 2048MB | 9.63s | 10.38 MB/s | 331MB |

**Note:** Test 3 was faster - possible variability in network/disk I/O. Average: ~12s, ~8.1 MB/s.
