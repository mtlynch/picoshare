# VM Management Alternatives to Vagrant

## Current Situation with Vagrant

### Problems Encountered

1. **SSH backgrounding issues** - Required workaround with `screen`
2. **Locking conflicts** - Multiple processes can conflict, requiring manual cleanup
3. **Slow rsync** - 2-4 minutes to sync 8.6GB of test files per VM creation
4. **Heavy resource usage** - Full VM provisioning overhead
5. **Transient failures** - Occasional unexplained failures requiring retry
6. **Complexity** - Many layers (Vagrant → vagrant-libvirt → libvirt → QEMU/KVM)

### What Works

1. **Memory isolation** - Can set exact RAM limits per test
2. **Clean state** - Each test starts fresh
3. **Port forwarding** - Easy access to PicoShare
4. **Box ecosystem** - Pre-built Ubuntu images

## Alternative Solutions

### 1. Firecracker (Recommended for Performance Testing)

**What it is:** Lightweight microVM technology from AWS, designed for serverless workloads.

**Pros:**
- ⚡ **Extremely fast startup** - ~125ms boot time (vs 3-4 minutes for Vagrant)
- 🪶 **Minimal overhead** - 5MB memory overhead per VM
- 🔒 **Strong isolation** - KVM-based like full VMs but optimized
- 📦 **Small footprint** - No complex provisioning layers
- 🎯 **Purpose-built** - Designed exactly for this use case (ephemeral, isolated workloads)
- 💰 **Resource efficient** - Can run hundreds of microVMs on one host

**Cons:**
- 🔧 **More manual setup** - No high-level tooling like Vagrant
- 📚 **Learning curve** - Different mental model than traditional VMs
- 🐧 **Linux kernel only** - Can't test on other OSes (but we only need Ubuntu)
- 🛠️ **Less mature tooling** - Would need to write more custom scripts
- ⚠️ **Root filesystem complexity** - Need to build custom root images

**Implementation effort:** Medium (2-3 days to build tooling)

**Impact on test time:**
- VM startup: 3-4 min → ~5 seconds
- File sync: Build into image once, no per-test sync
- **Total matrix time: 6 hours → ~1-2 hours**

**Code snippet:**
```bash
# Start Firecracker VM with memory limit
firecracker --api-sock /tmp/firecracker.sock \
  --config-file vm-config.json

# Config sets exact memory limit
{
  "machine-config": {
    "vcpu_count": 1,
    "mem_size_mib": 512
  }
}
```

### 2. Direct QEMU/KVM + libvirt (Recommended for Quick Win)

**What it is:** Use libvirt directly without Vagrant wrapper, same tech stack underneath.

**Pros:**
- ✅ **Minimal changes** - Already using this stack via Vagrant
- 🎯 **Remove Vagrant overhead** - Eliminate locking, sync issues
- 🛠️ **Mature tooling** - `virsh` CLI is stable and well-documented
- 📦 **Keep existing images** - Can reuse Vagrant boxes as base
- 🔧 **More control** - Direct access to all libvirt features
- 🚀 **Faster** - No Vagrant startup overhead (~30 sec savings)

**Cons:**
- ⚙️ **More manual scripting** - Need to handle what Vagrant did automatically
- 📝 **XML configuration** - libvirt uses XML domain definitions
- 🔌 **Manual networking** - Need to set up port forwarding ourselves
- 💾 **Image management** - Have to handle disk images manually

**Implementation effort:** Low (1 day to rewrite scripts)

**Impact on test time:**
- VM startup: 3-4 min → 2.5-3 min (modest improvement)
- File sync: Still need rsync or alternatives
- **Total matrix time: 6 hours → ~4-5 hours**

**Code snippet:**
```bash
# Create VM from XML definition
virsh create vm-2048mb.xml

# XML sets memory limit
<domain type='kvm'>
  <memory unit='MiB'>2048</memory>
  ...
</domain>
```

### 3. Docker/Podman (Not Recommended)

**What it is:** Container technology, shares kernel with host.

**Pros:**
- ⚡ **Instant startup** - Milliseconds vs minutes
- 💾 **Minimal overhead** - No full OS per container
- 📦 **Easy distribution** - Docker Hub ecosystem
- 🔄 **Quick rebuild** - Fast iteration during development

**Cons:**
- ❌ **Memory limits unreliable** - Cgroups memory limits aren't hard limits, can be exceeded
- ⚠️ **Not true isolation** - Shared kernel affects performance measurement
- 🔍 **Observability issues** - Can't measure memory usage accurately from inside container
- 🚫 **Not realistic** - Production PicoShare runs in VMs/bare metal, not containers
- 📊 **Skewed results** - Container overhead different than VM/bare metal

**Implementation effort:** Low (1 day)

**Verdict:** ❌ **Don't use** - Memory isolation is critical for our tests

### 4. systemd-nspawn (Not Recommended)

**What it is:** Lightweight container system built into systemd.

**Pros:**
- 🪶 **Lightweight** - Simpler than Docker
- 🔒 **Better isolation** - More VM-like than Docker containers
- 📦 **No daemon** - Part of systemd, always available

**Cons:**
- ❌ **Same memory issues as Docker** - Cgroups limits not hard
- 🔧 **Less tooling** - Fewer pre-built images
- 📚 **Less documentation** - Smaller community

**Implementation effort:** Medium (2 days)

**Verdict:** ❌ **Don't use** - Same fundamental issues as Docker

### 5. LXC/LXD (Possible Alternative)

**What it is:** System containers that are more VM-like than Docker.

**Pros:**
- ⚡ **Fast startup** - Seconds vs minutes
- 🔒 **Good isolation** - Can set hard memory limits
- 💾 **Efficient** - Less overhead than full VMs
- 🛠️ **Mature** - Well-established, used by Canonical
- 📦 **Image ecosystem** - Pre-built Ubuntu images

**Cons:**
- 🔧 **Still containers** - Shared kernel can affect results
- ⚙️ **Complex setup** - LXD daemon, storage pools, etc.
- 📊 **Memory measurement** - Need to verify accuracy vs VMs
- 🤔 **Uncertainty** - Need to validate memory behavior matches VMs

**Implementation effort:** Medium (2 days)

**Verdict:** ⚠️ **Maybe** - Need to validate memory isolation first

### 6. Cloud-init + libvirt Direct (Hybrid Approach)

**What it is:** Use libvirt directly but with cloud-init for provisioning.

**Pros:**
- ✅ **Keep VM isolation** - Real VMs, accurate memory limits
- 🚀 **Remove Vagrant** - Eliminate locking and overhead
- 📦 **Standard provisioning** - cloud-init is widely used
- 🔄 **Reusable images** - Build once, clone for tests

**Cons:**
- 📝 **YAML configuration** - cloud-init config files
- 🔧 **Manual networking** - Need to configure ourselves
- 💾 **Image building** - Need to create base images

**Implementation effort:** Medium (2 days)

**Verdict:** ✅ **Good option** - Best of both worlds

## Comparison Matrix

| Solution | Boot Time | Memory Isolation | Complexity | Test Time | Recommended |
|----------|-----------|------------------|------------|-----------|-------------|
| **Vagrant (current)** | 3-4 min | ✅ Excellent | High | 6 hours | ⚠️ Working but slow |
| **Firecracker** | ~5 sec | ✅ Excellent | Medium | 1-2 hours | ✅ **Best for perf** |
| **Direct libvirt** | 2.5-3 min | ✅ Excellent | Low-Medium | 4-5 hours | ✅ **Quick win** |
| **Docker/Podman** | <1 sec | ❌ Poor | Low | N/A | ❌ Don't use |
| **systemd-nspawn** | <1 sec | ❌ Poor | Medium | N/A | ❌ Don't use |
| **LXC/LXD** | 5-10 sec | ⚠️ Good | Medium | 2-3 hours | ⚠️ Need validation |
| **cloud-init + libvirt** | 2-3 min | ✅ Excellent | Medium | 4-5 hours | ✅ Good middle ground |

## Recommendations

### For Immediate Use (Next 24 Hours)

**Stick with Vagrant** - We just fixed the main issues:
- SSH backgrounding: Solved with `screen`
- Arithmetic bug: Fixed
- Locking: Added detection and cleanup

The tests are working now. Running the full matrix once will give us baseline data.

### For Short-term (Next Week)

**Switch to direct libvirt** - Remove Vagrant layer:
- Keep proven tech stack (KVM/libvirt)
- Eliminate Vagrant-specific issues
- ~30% time savings from removing overhead
- Low implementation risk

**Implementation plan:**
1. Convert Vagrantfile to libvirt XML domain template
2. Rewrite run-test to use `virsh` instead of `vagrant`
3. Pre-build VM image with test files included
4. Test with single run, then matrix

### For Long-term (Next Month)

**Migrate to Firecracker** - Optimal for performance testing:
- 60-80% time reduction for full matrix
- Minimal resource overhead
- Purpose-built for ephemeral workloads
- Can run many tests in parallel

**Implementation plan:**
1. Build custom root filesystem with PicoShare + test files
2. Create firecracker-runner script (similar to run-test)
3. Implement memory limit configuration
4. Add monitoring via Firecracker API
5. Test and validate results match VM tests

## Why Not Containers for Performance Testing?

**Memory limits in containers are soft limits:**

```bash
# Docker example - container can exceed limit temporarily
docker run --memory=512m myapp

# What actually happens:
# - Process can allocate >512MB briefly
# - OOM killer intervenes eventually
# - Timing is unpredictable
# - Defeats purpose of testing exact memory limits
```

**VM memory limits are hard limits:**
```bash
# VM example - cannot exceed limit
virsh setmaxmem vm 512M

# What actually happens:
# - Process tries to allocate >512MB
# - Kernel denies immediately
# - Predictable, deterministic behavior
# - This is what we need to test
```

## Implementation Recommendation

### Phase 1: Run Tests with Current Vagrant Setup (Today)
- Execute full matrix to get baseline data
- Document any issues that arise
- Proves the test methodology works

### Phase 2: Switch to Direct libvirt (This Week)
**Effort:** 1 day of development
**Benefit:** Eliminate Vagrant flakiness, 20-30% time savings

**Changes:**
```bash
# Before (Vagrant)
vagrant up
vagrant ssh -c "command"
vagrant destroy -f

# After (libvirt)
virsh create vm-from-template.xml
virsh exec vm-2048mb -- command
virsh destroy vm-2048mb && virsh undefine vm-2048mb
```

### Phase 3: Consider Firecracker (Later)
**Effort:** 2-3 days of development
**Benefit:** 60-80% time savings, can run tests continuously

**Decision factors:**
- How often will tests run? (Daily → Firecracker worth it)
- Do we need Windows testing? (No → Firecracker fine)
- Team comfort with new tech? (Low → stick with libvirt)

## Proof of Concept Scripts

### Direct libvirt (Minimal Changes)

```bash
#!/usr/bin/env bash
# run-test-libvirt - Drop-in replacement using libvirt directly

RAM_MB="$1"
FILE_SIZE="$2"

# Create VM from template with specific memory
VM_NAME="picoshare-test-$$"
sed "s/MEMORY_MB/${RAM_MB}/g" vm-template.xml > /tmp/${VM_NAME}.xml

# Start VM
virsh create /tmp/${VM_NAME}.xml

# Wait for IP (via DHCP lease)
VM_IP=$(virsh domifaddr "$VM_NAME" | awk '/ipv4/{print $4}' | cut -d/ -f1)

# Run PicoShare via SSH (same as Vagrant)
ssh "vagrant@${VM_IP}" "screen -dmS picoshare ./picoshare"

# Upload test file
curl -F "file=@${FILE_SIZE}.bin" "http://${VM_IP}:4001/api/entry"

# Cleanup
virsh destroy "$VM_NAME"
virsh undefine "$VM_NAME"
```

This is ~90% similar to what we have now, just removes Vagrant layer.

## Conclusion

**Short answer:** Switch to **direct libvirt** this week for quick wins, consider **Firecracker** later if testing becomes frequent.

**Vagrant is okay for now** because we've fixed the main issues, but it's adding unnecessary complexity and time. The rsync overhead alone (2-4 min per test) adds 80-160 minutes to the full matrix.

Direct libvirt gives us most of the benefits with minimal risk, while Firecracker is the optimal long-term solution if performance testing becomes a regular activity.
