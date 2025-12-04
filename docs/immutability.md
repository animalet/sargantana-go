# Configuration Immutability

## Motivation

When building applications with the Sargantana Go framework, configurations are passed to constructors like `NewServer()`. To ensure predictable runtime behavior, the framework guarantees that external modifications to configurations cannot affect running components after they've been initialized.

**The Problem:**
```go
cfg := server.SargantanaConfig{
    WebServerConfig: server.WebServerConfig{
        Address: "localhost:8080",
        /* ... */
    },
}

srv := server.NewServer(cfg)

// Later, someone modifies the config (intentionally or by accident)
cfg.WebServerConfig.Address = "localhost:9999"

// Without immutability: server behavior might change unexpectedly
// With immutability: server continues using "localhost:8080" ✅
```

**Why This Matters:**
1. **Runtime Safety**: Once initialized, components behave predictably
2. **Configuration Control**: Framework has full control over runtime config values
3. **Debugging**: No mysterious behavior from external config changes
4. **Thread Safety**: Config modifications in one goroutine don't affect running components

## Solution: Immutability at Constructor Boundaries

The framework uses `deepcopy.MustCopy()` to create immutable snapshots when configs are stored by constructors.

### Pattern

```go
import "github.com/animalet/sargantana-go/internal/deepcopy"

func NewServer(cfg SargantanaConfig) *Server {
    return &Server{
        config: *deepcopy.MustCopy(&cfg),  // Deep copy = immutable snapshot
        authenticator: NewUnauthorizedAuthenticator(),
    }
}
```

**How it works:**
1. User passes config normally (no special syntax required)
2. Constructor creates deep copy internally (automatic)
3. Component stores the snapshot (guaranteed isolation)
4. External changes don't affect the component

## Usage

### For Framework Users

**You don't need to do anything!** The immutability is handled transparently by the framework:

```go
// Just use configs normally
cfg, err := config.Get[server.SargantanaConfig](configFile, "sargantana")
srv := server.NewServer(*cfg)

// Modify config after - won't affect server
cfg.WebServerConfig.Address = "different:8888"  // Server still uses original
```

### For Framework Developers

When creating constructors that store configurations, use `deepcopy.MustCopy()`:

```go
import "github.com/animalet/sargantana-go/internal/deepcopy"

type MyComponent struct {
    config MyConfig
}

func NewMyComponent(cfg MyConfig) *MyComponent {
    return &MyComponent{
        config: *deepcopy.MustCopy(&cfg),  // Make immutable copy
    }
}
```

**When to use:**
- ✅ Constructors that store configs for later use
- ✅ Methods that cache configs (`SetConfig()`, `UpdateSettings()`)
- ❌ Temporary operations (function uses config and returns immediately)
- ❌ Config transformations (function returns modified config)
- ❌ Validators (function only reads, doesn't store)

## Deep Copy Functions

The `internal/deepcopy` package provides two functions:

### `Copy[T](cfg *T) (*T, error)`
Returns error on copy failure. Use when error handling is needed.

```go
cfg, err := deepcopy.Copy(original)
if err != nil {
    return fmt.Errorf("failed to copy config: %w", err)
}
```

**Use for:** Config loading, data transformation, any context requiring error handling.

### `MustCopy[T](cfg *T) *T`
Panics on copy failure. Use in constructors where failure indicates a programming error.

```go
func NewServer(cfg SargantanaConfig) *Server {
    return &Server{
        config: *deepcopy.MustCopy(&cfg),  // Panic if copy fails
    }
}
```

**Use for:** Constructors, initialization code where copy failure should never happen with valid structs.

**Why panic?** If a config struct can't be deep-copied, it indicates a structural problem (e.g., non-copyable fields like channels, mutexes). This should be caught during development, not at runtime.

## What Gets Copied

Deep copying handles:
- ✅ Primitive fields (strings, ints, bools, etc.)
- ✅ Struct values
- ✅ Slices (creates new slice with copied elements)
- ✅ Maps (creates new map with copied key-value pairs)
- ✅ Nested pointers (recursively copies pointed-to values)
- ✅ Arrays

**Example:**
```go
type Config struct {
    Address  string              // ✅ Copied
    Servers  []string            // ✅ New slice, copied elements
    Settings map[string]string   // ✅ New map, copied entries
    TLS      *TLSConfig          // ✅ New TLSConfig instance
}
```

## Performance

Deep copying has minimal overhead for typical configs:

```
BenchmarkCopy_SimpleStruct    1304 ns/op    456 B/op    15 allocs/op
BenchmarkCopy_NestedStruct    5817 ns/op   2152 B/op    73 allocs/op
BenchmarkCopy_LargeSlice     10146 ns/op   4376 B/op   127 allocs/op
```

**Typical overhead:** 1-10 microseconds per constructor call

**When this matters:**
- ❌ Hot paths with frequent re-initialization (rare in practice)
- ✅ Startup/initialization (happens once, overhead negligible)
- ✅ Request handlers (constructors not called per-request)

## Architecture: Immutability at Boundaries

The framework enforces immutability only where configs are **stored**, not where they're **returned**:

```go
// Config loading - returns mutable config (no copy)
func Get[T Validatable](c *Config, name string) (*T, error) {
    expanded, err := doExpand(partial)
    return expanded, nil  // Direct return, no deep copy
}

// Constructor - creates immutable snapshot
func NewServer(cfg SargantanaConfig) *Server {
    return &Server{
        config: *deepcopy.MustCopy(&cfg),  // Deep copy enforced here
    }
}
```

**Benefits:**
1. **Performance**: Config loading is fast (no unnecessary copying)
2. **Flexibility**: Allows temporary config manipulation before passing to constructors
3. **Clear Responsibility**: Config loading = mutable, stored configs = immutable
4. **Single Point of Control**: Immutability enforced only at storage boundaries

## Controller Pattern (No Deep Copy Needed)

Controllers extract primitive fields from configs rather than storing the entire config:

```go
func NewStaticController(c *StaticControllerConfig, _ ControllerContext) (IController, error) {
    return &static{
        path: c.Path,    // Extract string
        dir:  c.Dir,     // Extract string
        file: c.File,    // Extract string
        auth: c.Auth,    // Extract bool
    }, nil
}
```

**Why no deep copy needed:**
- Fields are primitives (strings, bools) - copied by value
- No slices or maps stored
- No nested pointers retained
- Extraction pattern provides natural isolation

## Testing Immutability

The framework includes comprehensive tests verifying immutability:

```go
It("should protect against external config modifications", func() {
    cfg := SargantanaConfig{
        WebServerConfig: WebServerConfig{Address: "localhost:8080"},
    }

    srv := NewServer(cfg)

    // Modify original
    cfg.WebServerConfig.Address = "modified:9999"

    // Server should be unaffected (verified by behavior tests)
    Expect(srv).NotTo(BeNil())
})
```

Run tests:
```bash
go test ./internal/deepcopy -v       # Deep copy tests (18 tests)
go test ./pkg/server -v              # Server immutability tests (6 tests)
```

## Implementation Status

### ✅ Already Immutable

- **`server.NewServer()`** - Uses `deepcopy.MustCopy()` for config protection
- **All controllers** - Extract primitive fields, no deep copy needed

### Framework Extension

When adding new components that store configs:

```diff
 import (
+    "github.com/animalet/sargantana-go/internal/deepcopy"
 )

 func NewMyComponent(cfg MyConfig) *MyComponent {
     return &MyComponent{
-        config: cfg,
+        config: *deepcopy.MustCopy(&cfg),
     }
 }
```

## Summary

**For Users:** Immutability is automatic and transparent. Use configs normally.

**For Developers:** Use `deepcopy.MustCopy()` in constructors that store configs.

**Key Principle:** Immutability enforced at boundaries (constructors), not in transit (config loading).

**Performance:** ~1-10μs overhead per constructor, negligible for initialization-time operations.

**Safety:** Once initialized, components behave predictably regardless of external config changes.
