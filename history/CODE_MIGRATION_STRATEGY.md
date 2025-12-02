# Code Migration Strategy: onboard/ → api/

## Analysis: What Code Can Be Shared?

### Onboard-Specific (Stays in `onboard/`)
- ✅ `internal/diarization/` - Diarization subprocess integration
- ✅ `internal/fetch/` - Fetching audio from Limitless API
- ✅ `internal/docker/` - Docker lifecycle management
- ✅ `internal/server/handlers.go` - Job submission handlers (onboard-specific)
- ✅ `cmd/server/main.go` - Onboarding server entry point

### Potentially Shared (Could Move to `api/`)
- ⚠️ `internal/export2elastic/` - Exporting to Elasticsearch
  - **Analysis**: Currently used by onboard for importing diarization results
  - **Decision**: Could be useful for api/ if we need to import data, but probably stays in onboard/
  - **Recommendation**: Keep in onboard/ for now, move later if needed

### Already Shared
- ✅ `storage/` - Already a shared package (used by both)
- ✅ `cmd/diarize/` - Python diarization (used by onboard)

## Migration Strategy

### Phase 1: Create `api/` with Stubs

**Action**: Set up `api/` directory structure with minimal working code

**Files to create**:
- `api/go.mod`
- `api/cmd/server/main.go` - Basic server that starts
- `api/internal/server/server.go` - Server struct with storage initialization
- `api/internal/server/handlers.go` - Stub handlers (health check, etc.)

**Goal**: Get `hai-api` binary working and able to start

### Phase 2: Identify Shared Patterns

**As we build `api/`, identify patterns from `onboard/` that could be reused:**

1. **Storage initialization pattern**:
   ```go
   // onboard/internal/server/handlers.go (lines 43-58)
   esURL := os.Getenv("ELASTICSEARCH_URL")
   esStorage, err := storage.NewElasticsearchStorage(esURL)
   ```
   - **Action**: Create helper in `api/internal/server/storage.go`
   - **Benefit**: Consistent storage setup

2. **Error handling patterns**:
   - JSON error responses
   - HTTP status code handling
   - **Action**: Create `api/internal/server/errors.go` with helpers

3. **Server setup patterns**:
   - Router configuration
   - Middleware setup
   - **Action**: Create `api/internal/server/router.go` with setup helpers

### Phase 3: Extract Shared Code (If Needed)

**If we find code that should be shared:**

**Option A: Move to `api/` and have `onboard/` import from `api/`**
- Pros: `api/` becomes the "main" server, onboard uses it
- Cons: Creates dependency: `onboard/` → `api/` (onboard depends on api)

**Option B: Move to shared package (e.g., `internal/shared/` or `pkg/`)**
- Pros: No circular dependencies, both can use it
- Cons: Another package to maintain

**Option C: Keep in `onboard/`, copy patterns to `api/`**
- Pros: No dependencies, onboard stays independent
- Cons: Some code duplication (but patterns, not exact code)

**Recommendation**: **Option C** (copy patterns) for now, because:
- Most code is actually onboard-specific
- `storage/` package already provides the main shared functionality
- Keeps `onboard/` independent
- Can refactor later if we find significant duplication

### Phase 4: Incremental Migration (If Applicable)

**If we decide to move code from `onboard/` to `api/`:**

1. **Move code** from `onboard/internal/X/` to `api/internal/X/`
2. **Update `onboard/`** to import from `api/`:
   ```go
   // onboard/internal/server/handlers.go
   import "hai/api/internal/server" // or whatever we moved
   ```
3. **Test** that `onboard/` still works
4. **Remove** old code from `onboard/` once migration complete

## Recommended Approach

### Start Fresh, Copy Patterns ✅ **DECISION: Using This Approach**

**Best approach for now:**

1. **Create `api/` with new code** (don't copy from onboard initially)
2. **Reference `onboard/` for patterns** (how they initialize storage, handle errors, etc.)
3. **Keep `onboard/` unchanged** (it works, don't break it)
4. **If we find significant duplication later**, then extract to shared package

**Status**: ✅ Approved - Will use this approach for implementation

**Why this works:**
- ✅ `onboard/` stays independent and working
- ✅ `api/` is built fresh with best practices
- ✅ No risky migrations of working code
- ✅ Can refactor later if needed

### What to Copy/Reference from `onboard/`

**Patterns to reference (not copy exactly):**

1. **Storage initialization** (`onboard/internal/server/handlers.go:43-58`):
   - How to initialize Elasticsearch storage
   - Error handling if ES unavailable
   - **Use in**: `api/internal/server/server.go`

2. **Server struct pattern** (`onboard/internal/server/handlers.go:25-32`):
   - Server struct with storage field
   - **Use in**: `api/internal/server/server.go`

3. **Handler patterns**:
   - JSON encoding/decoding
   - Error responses
   - **Use in**: `api/internal/server/handlers.go`

## File Structure After Migration (If We Do It)

```
hai/
├── onboard/              # Onboarding (mostly unchanged)
│   ├── internal/
│   │   ├── diarization/ # Stays (onboard-specific)
│   │   ├── fetch/       # Stays (onboard-specific)
│   │   ├── docker/      # Stays (onboard-specific)
│   │   └── server/      # Stays (onboard-specific handlers)
│   └── cmd/server/      # Stays
│
├── api/                  # New API server
│   ├── internal/
│   │   ├── server/      # New server code (patterns from onboard)
│   │   ├── contacts/    # New contacts code
│   │   └── handlers/    # New API handlers
│   └── cmd/server/      # New API server entry point
│
└── storage/              # Shared (already exists)
```

## Decision: What Actually Gets Moved?

**Short answer: Probably nothing initially.**

**Reasoning:**
- `onboard/` code is mostly onboarding-specific
- `storage/` package already provides shared functionality
- Better to build `api/` fresh with lessons learned
- Can extract shared code later if duplication becomes significant

**Exception: If `export2elastic/` becomes useful for `api/`:**
- Could move to `api/internal/export/` or shared package
- But probably not needed initially

## Implementation Plan

### Step 1: Create `api/` Structure
- Set up directories
- Create `go.mod`
- Create basic server that starts

### Step 2: Build API Server
- Reference `onboard/` for patterns (storage init, error handling)
- Don't copy code, just use as reference
- Build fresh with best practices

### Step 3: Test Both Servers
- `hai-onboard` still works (unchanged)
- `hai-api` works (new code)

### Step 4: Future Refactoring (If Needed)
- If we find significant shared code, extract to shared package
- Or move to `api/` and have `onboard/` import it
- But don't do this prematurely

## Benefits of This Approach

1. ✅ **Low risk**: Don't break working `onboard/` code
2. ✅ **Clean start**: Build `api/` with best practices
3. ✅ **No premature optimization**: Extract shared code only if needed
4. ✅ **Independent**: Both servers can evolve independently
5. ✅ **Flexible**: Can refactor later if patterns emerge

