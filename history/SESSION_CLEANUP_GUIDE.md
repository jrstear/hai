# End-of-Session Cleanup Guide

## Standard Cleanup Process

### 1. Check Current Status
```bash
git status                    # See what's changed
git diff --stat               # Summary of changes
bd list --status in_progress  # Check for unfinished work
```

### 2. Sync Beads Issues
```bash
bd sync                       # Flush beads changes to .beads/issues.jsonl
# If no upstream: beads will auto-sync on commit
```

### 3. Review Changes
- Check for sensitive files (`.env`, API keys)
- Review file organization (planning docs in `history/`)
- Ensure documentation references are in place

### 4. Stage and Commit
```bash
# Stage all changes (respects .gitignore)
git add .

# Or stage selectively:
git add history/              # Planning docs
git add cmd/diarize/          # Tool code
git add .beads/issues.jsonl   # Issue tracking state
git add AGENTS.md SETUP.md    # Root docs

# Review what will be committed
git status

# Commit with descriptive message
git commit -m "Brief summary of changes

- Specific change 1
- Specific change 2
- References: issue IDs if applicable"
```

### 5. Push (if remote exists)
```bash
git push
# Or if first time: git push -u origin main
```

### 6. Verify Clean State
```bash
git status                    # Should be clean
bd ready --json              # See what's next
```

## What to Commit

### ✅ Always Commit:
- Code changes (`cmd/`, scripts)
- Documentation (`AGENTS.md`, `SETUP.md`, `seed.md`)
- Planning docs (`history/`)
- Beads issues (`.beads/issues.jsonl`)
- Configuration (`.gitignore`, `.gitattributes`)

### ❌ Never Commit:
- `.env` files (API keys, secrets)
- `.DS_Store` (macOS system files)
- `data/audio/` (large binary files, if in .gitignore)
- `*.db` files (SQLite databases, if temporary)
- Build artifacts, cache files

## Session Summary Template

Before committing, create a mental or written summary:

### Today's Accomplishments:
- [ ] What was completed?
- [ ] What documentation was created?
- [ ] What issues were created/updated?
- [ ] What decisions were made?

### Next Session:
- [ ] What's ready to work on? (`bd ready`)
- [ ] What blockers exist?
- [ ] What's the priority?

## Quick Checklist

- [ ] All code saved
- [ ] Documentation references added to beads issues
- [ ] Planning docs organized in `history/`
- [ ] Beads synced (`bd sync`)
- [ ] `.gitignore` checked (no secrets committed)
- [ ] Changes reviewed (`git status`, `git diff`)
- [ ] Committed with descriptive message
- [ ] Pushed to remote (if applicable)
- [ ] Clean state verified

