# PicoShare Performance Test Results

**Date**: 2026-01-28 21:25
**Test Configuration**: 100MB file upload, 2048MB RAM VM
**Infrastructure**: Vagrant + libvirt on Debian 12 (vagrant-libvirt 0.12.2-4)

## Issue Resolved

**Root Cause**: The Vagrantfile was excluding `test-files/` directory from rsync, preventing test files from being available in the VM for uploads.

**Fix Applied**: Removed `test-files/` from the rsync exclusion list in Vagrantfile:

```ruby
# Before:
config.vm.synced_folder ".", "/vagrant", type: "rsync", rsync__exclude: [".git/", "test-files/", ".vagrant/"]

# After:
config.vm.synced_folder ".", "/vagrant", type: "rsync", rsync__exclude: [".git/", ".vagrant/"]
```

## Test Results

### Manual Test (Verified Working)

```
✅ PASS: 100M upload with 2048MB RAM
   File Size: 100MB
   VM Memory: 2048MB
   Upload Time: 8.74 seconds
   Throughput: 11.44 MB/s
   HTTP Status: 200
```

### Test Environment

- **VM**: Ubuntu 22.04 (generic/ubuntu2204)
- **Memory**: 2048MB RAM
- **CPU**: 1 core (AMD EPYC with KVM)
- **Network**: Port forward 4001 (guest) → 4001 (host)
- **File Sync**: rsync
- **PicoShare Version**: perf-test-7f6d07c

### Steps Executed

1. ✅ Built PicoShare binary
2. ✅ Started VM with 2GB RAM
3. ✅ Synced test files to VM (8.6GB total)
4. ✅ Started PicoShare server in VM
5. ✅ Authenticated via API
6. ✅ Uploaded 100MB file successfully
7. ✅ Received HTTP 200 response

## Conclusion

The performance test infrastructure is now fully functional. The vagrant-libvirt networking issue has been resolved (v0.12.2-4), and the file sync bug has been fixed. Test successfully completed with good throughput (11.44 MB/s for 100MB upload).

The test matrix can now be expanded to include:

- Multiple file sizes: 100MB, 500MB, 1GB, 2GB, 5GB
- Multiple memory configurations: 256MB, 512MB, 1024MB, 2048MB
