# Performance Improvements for Large Files (Issue #355)

## Problem Summary
PicoShare has severe performance issues with large files:
- 7GB file upload took 10 hours on high-end hardware
- 500 KB/s download speeds for 6GB file over LAN
- Root cause: Files stored as ~320KB chunks in SQLite, requiring ~22,000 separate queries to read a 7GB file

---

## Independent Improvements

Each improvement can be implemented and tested separately.

### 1. Batch Chunk Reading (Download Speed)

**Problem**: `populateBuffer()` in `reader.go:85-114` fetches ONE chunk per query.

**Solution**: Fetch multiple chunks in a single query, cache them for sequential reads.

**File**: `store/sqlite/file/reader.go`

**Changes**:
- Add `batchSize` field (default: 10 chunks = ~3.2MB per batch)
- Add `chunkCache [][]byte` and `cacheStartIdx int64` fields
- Replace single-chunk query with batch query: `WHERE id=? AND chunk_index >= ? AND chunk_index < ?`
- Invalidate cache on `Seek()`

**Expected impact**: 10x fewer queries for downloads

---

### 2. File Size Denormalization (Metadata Speed)

**Problem**: Every metadata query runs `SUM(LENGTH(chunk))` across all chunks (`entries.go:28-34`, `entries.go:112-118`).

**Solution**: Store file size in `entries` table during upload.

**Files**:
- `store/sqlite/migrations/016-add-file-size-column.sql` (new)
- `store/sqlite/entries.go`

**Migration**:
```sql
ALTER TABLE entries ADD COLUMN file_size INTEGER;

UPDATE entries SET file_size = (
    SELECT SUM(LENGTH(chunk)) FROM entries_data WHERE entries_data.id = entries.id
);
```

**Query changes**: Use `COALESCE(entries.file_size, <subquery>)` for backward compatibility with unmigrated rows.

**Expected impact**: Metadata queries O(1) instead of O(num_chunks)

---

### 3. Larger Chunk Size (Upload/Download Speed)

**Problem**: 320KB chunks create excessive row overhead (7GB = 22,000 rows).

**Solution**: Increase default chunk size to 4MB.

**File**: `store/sqlite/sqlite.go:16`

**Change**:
```go
// Before
const defaultChunkSize = uint64(32768 * 10)  // ~320KB

// After
const defaultChunkSize = uint64(4 * 1024 * 1024)  // 4MB
```

**Backward compatibility**: `getChunkSize()` in `reader.go:140-159` already reads actual chunk size from DB, so existing files continue to work.

**Expected impact**: 12x fewer rows per file, fewer INSERT/SELECT operations

---

### 4. SQLite Connection Pool Tuning

**Problem**: No explicit connection pool configuration in `sqlite.go`.

**Solution**: Add explicit limits for SQLite's single-writer model.

**File**: `store/sqlite/sqlite.go` (after line 38)

**Change**:
```go
ctx.SetMaxOpenConns(1)    // SQLite is single-writer
ctx.SetMaxIdleConns(1)    // Keep connection warm
ctx.SetConnMaxLifetime(0) // Don't close idle connections
```

**Expected impact**: Reduced connection churn overhead

---

### 5. Add Index for Download Tracking

**Problem**: `downloads` table has no index on `entry_id` for history queries.

**Solution**: Add index via migration.

**File**: `store/sqlite/migrations/017-add-downloads-index.sql` (new)

```sql
CREATE INDEX idx_downloads_entry_id ON downloads (entry_id);
```

**Expected impact**: Faster download history lookups for files with many downloads
