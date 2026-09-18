# RiftKV

A small Go-based Redis-like key-value server built as a learning project and prototype datastore.

This repository is currently in an early prototype stage: it has a working TCP server, RESP command parsing, in-memory key/value storage, TTL support, and a WAL-backed persistence layer. It is not yet a production-grade database engine.

---

## Project goal

RiftKV aims to explore the core concepts behind a simple in-memory database with durable writes:

- TCP server and client communication
- RESP command parsing
- KV command handling
- in-memory storage
- TTL-based expiration
- append-only WAL persistence
- restart recovery from WAL

The project is intentionally compact and educational rather than feature-complete.

---

## Current status

As of the current repository state, RiftKV is best described as:

- a working prototype
- a server with minimal command support
- a learning project for Go + persistence + DB internals
- not yet complete enough for production reliability or broad Redis compatibility

### Status summary

- TCP server: implemented
- RESP parsing: partially implemented
- basic KV commands: implemented
- TTL semantics: partially implemented
- persistence via WAL: partially implemented
- restart recovery: partially implemented
- concurrency protections: partial
- test coverage: minimal
- graceful shutdown: not implemented

---

## System design overview

RiftKV follows a simple database architecture:

Client
  ↓
TCP socket
  ↓
RESP parser
  ↓
command dispatcher
  ↓
Store
  ↓
WAL persistence

### In plain terms

1. A client connects to the server over TCP.
2. The server reads RESP-formatted commands from the socket.
3. The command is decoded into a Go structure.
4. A command handler executes the action.
5. The store updates its in-memory map.
6. For writes, a WAL record is appended to disk.
7. On restart, the WAL is replayed to rebuild the in-memory state.

---

## Architecture

### 1. Server layer: main.go

The server is implemented in [main.go](main.go).

Responsibilities:
- bind to TCP port 127.0.0.1:6380
- accept incoming client connections
- spawn a goroutine per connection
- parse RESP requests
- dispatch command names
- call into the store layer
- return RESP responses to clients

Core flow:
- `main()` creates a `Store` using `NewStore("riftkv.wal")`
- starts the TCP listener
- accepts connections in a loop
- each connection is handled by `handleConnection(conn, store)`

Current supported commands:
- `SET`
- `GET`
- `DEL`
- `PING`
- `EXISTS`

### 2. RESP parsing: readRESP.go

The RESP decoder is implemented in [readRESP.go](readRESP.go).

Responsibilities:
- read raw bytes from the socket
- decode RESP primitives into Go values
- support the basic RESP types used by the project

Supported types in the current code:
- simple strings (`+`)
- errors (`-`)
- integers (`:`)
- bulk strings (`$`)
- arrays (`*`)

Important functions:
- `readRESP(reader *bufio.Reader)`
- `readLine(reader *bufio.Reader)`
- `readBulkString(reader *bufio.Reader)`
- `readArray(reader *bufio.Reader)`

This is a lightweight parser suited to the project’s current needs, but it is not a fully robust RESP implementation.

### 3. Storage layer: store.go

The in-memory store and write path live in [store.go](store.go).

Main responsibilities:
- hold the current key-value data in memory
- guard access with a mutex
- persist writes to the WAL
- reload WAL entries on startup
- expire keys based on TTL

Important types:

```go
type Entry struct {
    value      string
    expiration time.Time
}

type Store struct {
    mu   sync.RWMutex
    data map[string]Entry
    wal  *WAL
}
```

Main functions:
- `NewStore(filename string)`
- `Set(key, value string, ttl time.Duration) error`
- `Get(key string) (string, bool)`
- `Delete(key string) error`
- `Exists(key string) bool`
- `applySet(key, value string, expiration int64)`
- `applyDelete(key string)`
- `ExpireLoop()`

### 4. Persistence layer: wal.go

The WAL is implemented in [wal.go](wal.go).

Responsibilities:
- open and maintain the WAL file
- append mutation records
- flush writes to disk via `Sync`
- replay WAL entries to rebuild state

Important types:

```go
type WALRecord struct {
    Operation  string `json:"operation"`
    Key        string `json:"key"`
    Value      string `json:"value"`
    Expiration int64  `json:"expiration"`
}

type WAL struct {
    file *os.File
}
```

Important functions:
- `OpenWAL(filename string)`
- `Append(record WALRecord) error`
- `Replay(store *Store) error`
- `Close() error`

The WAL is append-only JSON, one record per line.

### 5. Example client: client/main.go

The client in [client/main.go](client/main.go) is a rudimentary TCP client used for manual testing.

Responsibilities:
- open a connection to the server
- encode user input as RESP arrays
- send request bytes to the server
- print the response

This is not a production-ready client library; it is simply a small sample for interaction and debugging.

---

## What is implemented today

### Implemented features

- TCP listener and client connections
- RESP array parsing for basic commands
- `SET` command handling
- `GET` command handling
- `DEL` command handling
- `EXISTS` command handling
- `PING` command handling
- in-memory map-based key storage
- TTL storage representation
- WAL append/write flow
- WAL replay on startup
- expiration cleanup during runtime

### Partially implemented features

- RESP robustness and edge-case handling
- WAL integrity and crash recovery guarantees
- concurrency safety under multi-client writes
- TTL persistence semantics
- error handling and bad-input recovery
- file lifecycle management
- testing coverage
- graceful shutdown

---

## Persistence design

RiftKV uses a WAL model rather than snapshots.

### Write flow

When a write command succeeds:

1. the store builds a `WALRecord`
2. the WAL appends JSON to the file
3. the file is synced to disk
4. the entry is inserted into the in-memory map

This is intended to provide durability before the memory update is considered complete.

### Recovery flow

On startup:

1. `OpenWAL` opens or creates the WAL file
2. `NewStore` calls `wal.Replay(store)`
3. the WAL is read from the beginning
4. each record is decoded and applied to the in-memory map

The store is intentionally reconstructed from the WAL rather than snapshotting memory state.

---

## TTL behavior

TTL support is present, but is still simple.

### Representation

- in memory: `Entry.expiration time.Time`
- in WAL: `Expiration int64` as Unix milliseconds

### Command syntax used by the current server

The server accepts:

```text
SET key value EX seconds
```

Example:

```text
SET session abc123 EX 60
```

### Check strategy

Expiration is checked in two ways:

1. on `Get`
2. in background expiration loop

The expiration loop continuously scans the map and removes expired keys.

This means TTL works while the process is running, but the design is still relatively simple and not production hardened.

---

## Concurrency model

### What exists

The store uses a `sync.RWMutex`:

```go
var mu sync.RWMutex
```

or, in the current structure:

```go
type Store struct {
    mu   sync.RWMutex
    data map[string]Entry
    wal  *WAL
}
```

This protects the in-memory map from concurrent reads and writes.

### Current limitation

The WAL file itself is not protected by an equivalent synchronization strategy. Multiple client goroutines can call `Store.Set` and `Store.Delete` concurrently, and the WAL append path is not serialized by a dedicated WAL mutex.

This makes the project only partially concurrency-safe.

---

## Current risks and limitations

### 1. WAL safety

The WAL file is shared across goroutines, but there is no explicit lock protecting append operations. That means concurrent writes could produce ordering or interleaving issues.

### 2. Partial final record handling

The replay loop reads until `io.EOF`, but it does not explicitly detect or recover from a truncated last record. A final incomplete JSON line may be skipped silently.

### 3. Corrupt WAL handling

A single corrupt WAL record causes replay to fail and startup to abort.

### 4. No graceful shutdown

There is no signal handling or shutdown channel to clean up the listener, store, or WAL file.

### 5. No transactional boundary

The code writes to WAL and then mutates memory, but not under a single coordinated transaction model.

### 6. No snapshotting

The WAL can grow unbounded over time.

### 7. Minimal testing

There are only a few small tests and little coverage of failure modes or concurrency.

---

## Commands currently supported

### SET

Syntax:

```text
SET key value
SET key value EX seconds
```

Behavior:
- stores the value in memory
- writes a WAL record
- optionally attaches TTL

### GET

Syntax:

```text
GET key
```

Behavior:
- returns the value if present and not expired
- returns `$-1\r\n` if missing

### DEL

Syntax:

```text
DEL key
```

Behavior:
- deletes the key from memory
- writes a DEL record to the WAL
- returns `:1` if deleted, `:0` if not found

### EXISTS

Syntax:

```text
EXISTS key
```

Behavior:
- returns `:1` if present
- returns `:0` otherwise

### PING

Syntax:

```text
PING
```

Behavior:
- returns `+PONG\r\n`

---

## Build and run

### Go project structure

This repository is a Go module:

- `main.go`
- `store.go`
- `wal.go`
- `readRESP.go`
- `client/main.go`
- `store_test.go`

### Run the server

From the repository root:

```bash
go run .
```

The server binds to:

```text
127.0.0.1:6380
```

---

## Example interaction

### Example RESP request

```text
*3
$3
SET
$5
hello
$5
world

```

This corresponds to:

```text
SET hello world
```

### Example response

```text
+OK
```

---

## What is missing before this becomes a serious database

### Required for reliability

- WAL append serialization
- robust crash recovery / partial log handling
- deterministic replay semantics
- graceful shutdown
- corruption recovery policies
- stricter protocol validation

### Important engineering work

- snapshotting and compaction
- config-driven settings
- observability and metrics
- more tests for edge cases and concurrency
- clean package boundaries

### Future work beyond this stage

- replication
- sharding
- data eviction policies
- better data structures
- benchmarking and tuning
- backup/restore tooling

---

## Recommended interpretation of the repo

This project is best understood as:

> a compact, educational implementation of a Redis-like storage engine with WAL durability, not yet a production-grade database.

The repository already demonstrates the core building blocks of a simple DB, but the next focus should be reliability and correctness rather than more Redis feature breadth.

---

## Summary

RiftKV is a small but meaningful prototype that demonstrates:

- Go networking
- RESP parsing
- in-memory KV logic
- TTL expiration
- WAL-based persistence
- restart recovery from logs

It is a strong learning project and a good base for continued database engineering work, but it still needs significant hardening before it can be considered dependable as a datastore.
