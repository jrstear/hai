# Public Repo: Separating Elasticsearch from Onboarding

## Goal

Create a public repository (`hai-pub`) that shares the onboarding functionality (API fetching, diarization) without exposing Elasticsearch integration work, which is still in progress with schema changes.

## Exclusions from Public Repo

- `history/` - Planning documents
- `seed.*` - Seed files
- `.beads/` - Issue tracking
- `storage/` package - Elasticsearch implementation
- Elasticsearch-dependent CLI tools

## Three Options Considered

### Option 1: Build Tags (RECOMMENDED)

**Complexity:** Medium (2-3 hours)

**Approach:**
- Use Go build tags (`//go:build elasticsearch`) to conditionally compile ES code
- Create stub implementations for `!elasticsearch` builds
- Keep handlers ES-agnostic (already mostly done)
- Update UI to conditionally show/hide ES columns

**Pros:**
- Single codebase, no duplication
- Clean separation
- Easy to test both variants
- Can add ES later by building with tag

**Cons:**
- Requires build tag discipline
- Slightly more complex build process

**Implementation Tasks:**
1. Add build tags to `onboard/internal/export2elastic/` package
2. Create stub implementations for non-ES builds
3. Add build tags to ES-dependent CLI tools (`load-es`, `map-speakers`)
4. Update `handlers.go` to handle missing ES gracefully (mostly done)
5. Update UI templates to conditionally hide ES columns
6. Update `go.mod` to make ES dependencies optional
7. Update documentation to explain build tags
8. Test both build variants

### Option 2: Separate Packages

**Complexity:** Low (1 hour)

**Approach:**
- Create `onboard/internal/export2elastic_stub/` with no-op implementations
- Use build tags to switch between real and stub
- Or: just don't include `export2elastic/` in public repo

**Pros:**
- Very simple
- Clear separation

**Cons:**
- Some duplication
- Harder to maintain two codebases

**Implementation Tasks:**
1. Create stub package with no-op implementations
2. Update imports to use stub in public builds
3. Exclude real ES package from public repo

### Option 3: Feature Flag at Runtime (Simplest)

**Complexity:** Very Low (30 minutes)

**Approach:**
- Keep current code (already checks `ELASTICSEARCH_URL`)
- For public repo: just don't include ES-related files
- Remove ES imports from `handlers.go`
- Update UI to hide ES columns

**Pros:**
- Simplest
- No build tags needed
- Works immediately

**Cons:**
- Requires maintaining two versions of `handlers.go`
- Or: stub out the ES calls

**Implementation Tasks:**
1. Remove ES imports from `handlers.go`
2. Stub out ES calls or remove ES code paths
3. Exclude ES packages from public repo
4. Update UI to hide ES columns
5. Update documentation

## Decision

**Selected: Option 1 (Build Tags)**

Reason: Best long-term maintainability with single codebase. Allows public repo to be built without ES while keeping private repo with full functionality.

## Files to Exclude from Public Repo

- `history/` directory
- `seed.*` files
- `.beads/` directory
- `storage/` package (entire directory)
- `onboard/cmd/load-es/` (or build-tag it)
- `onboard/cmd/map-speakers/` (or build-tag it)
- `onboard/internal/export2elastic/` (or build-tag it)

## Files to Include in Public Repo

- `onboard/cmd/server/` - Main onboarding server
- `onboard/internal/diarization/` - Diarization integration
- `onboard/internal/fetch/` - API fetching
- `onboard/internal/server/` - HTTP handlers (ES parts build-tagged)
- `onboard/scripts/` - Setup scripts
- `onboard/templates/` - UI templates (ES parts conditionally hidden)
- `cmd/diarize/` - Standalone diarization tool
- `DEVELOPER_SETUP.md`, `SETUP.md` - Updated to remove ES references
- `onboard/README.md` - Updated for public version
- `onboard/Taskfile.yml` - Build configuration
- `onboard/go.mod` - Module definition (ES deps optional)

## Build Tag Strategy

### Files with Build Tags

1. **`onboard/internal/export2elastic/*.go`**
   - All files: `//go:build elasticsearch`
   - Stub file: `//go:build !elasticsearch`

2. **`onboard/cmd/load-es/main.go`**
   - `//go:build elasticsearch`

3. **`onboard/cmd/map-speakers/main.go`**
   - `//go:build elasticsearch`

### Build Commands

**Private repo (with ES):**
```bash
go build -tags elasticsearch ./onboard/cmd/server
```

**Public repo (without ES):**
```bash
go build ./onboard/cmd/server
```

## UI Updates

- Hide "Load Elasticsearch" column when built without ES
- Or show it but mark as "not available" (simpler)
- Update status display logic

## Documentation Updates

- Update `DEVELOPER_SETUP.md` to remove ES setup steps
- Update `SETUP.md` to remove ES references
- Update `onboard/README.md` for public version
- Add note about build tags if needed

## Testing Strategy

1. Build without ES tag - verify core functionality works
2. Build with ES tag - verify ES integration works
3. Test UI in both modes
4. Verify no ES dependencies leak into public build

## Task Dependencies

**Created beads for Option 1 implementation:**

1. **hai-9ob** - Add build tags to export2elastic package (FOUNDATION)
2. **hai-8q6** - Add build tags to ES-dependent CLI tools (depends on: hai-9ob)
3. **hai-duv** - Update handlers.go to gracefully handle missing ES (depends on: hai-9ob)
4. **hai-yom** - Update UI templates to conditionally hide ES columns (depends on: hai-duv)
5. **hai-wx5** - Update go.mod to make ES dependencies optional (depends on: hai-9ob)
6. **hai-19v** - Update documentation for public repo version (depends on: hai-9ob, hai-8q6, hai-duv)
7. **hai-k9s** - Test both build variants (depends on: hai-9ob, hai-8q6, hai-duv, hai-yom, hai-wx5)
8. **hai-2b5** - Create public repo structure and copy files (depends on: hai-k9s, hai-19v)

**Execution order:**
- Start with hai-9ob (foundation)
- Then hai-8q6 and hai-duv (can be parallel)
- Then hai-yom and hai-wx5 (can be parallel)
- Then hai-19v
- Then hai-k9s (testing)
- Finally hai-2b5 (public repo creation)

