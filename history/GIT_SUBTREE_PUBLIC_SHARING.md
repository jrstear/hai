# Git Subtree Strategy for Public Code Sharing

## Overview

You want to share a subset of your private repository publicly while keeping the ability to sync updates. Git subtree is an excellent approach for this.

## What is Git Subtree?

Git subtree allows you to:
- Extract a subdirectory from your private repo
- Push it to a public repo
- Keep the ability to pull updates from public back to private
- Push updates from private to public

## Recommended Structure

### Private Repo (Current)
```
hai/ (private)
├── cmd/
│   ├── onboard/          # ✅ Share publicly
│   ├── diarize/          # ✅ Share publicly
│   └── ingest/           # ✅ Share publicly
├── internal/             # ✅ Share publicly (if created)
├── data/                 # ❌ Keep private (personal data)
├── history/              # ❌ Keep private (planning docs)
├── .beads/               # ❌ Keep private (issue tracking)
└── seed.md               # ❌ Keep private (personal notes)
```

### Public Repo (To Create)
```
hai-public/ (public)
├── cmd/
│   ├── onboard/
│   ├── diarize/
│   └── ingest/
├── internal/
├── README.md
├── LICENSE
├── go.mod
└── .gitignore
```

## Setup Process

### Step 1: Create Public Repository

```bash
# On GitHub, create new public repo: hai-public
# Don't initialize with README (we'll push to it)
```

### Step 2: Prepare Private Repo for Sharing

```bash
cd /Users/jrstear/mine/git/hai

# Create a script to help with subtree operations
cat > scripts/sync-to-public.sh << 'EOF'
#!/bin/bash
# Sync selected directories to public repo

PUBLIC_REPO="git@github.com:yourusername/hai-public.git"
PUBLIC_BRANCH="main"

# Add public repo as remote (if not already added)
git remote add public $PUBLIC_REPO 2>/dev/null || true

# Push subtree to public repo
git subtree push --prefix=cmd public $PUBLIC_BRANCH
EOF

chmod +x scripts/sync-to-public.sh
```

### Step 3: Initial Push to Public

```bash
# Push cmd/ directory to public repo
git subtree push --prefix=cmd public main

# Or if you want to include more:
git subtree push --prefix=. public main --ignore-joins
# Then manually remove private files from public repo
```

### Step 4: Create .gitignore for Public

Create a file that will be used in public repo:

```bash
# .gitignore.public
# Environment variables and secrets
.env
.env.local
*.key
*.pem

# Personal data
data/
history/
.beads/
seed.md
*.ogg
*.mp3
*.wav

# macOS
.DS_Store

# Python
__pycache__/
*.pyc
venv/

# IDE
.vscode/
.idea/
```

## Better Approach: Use a Public Subdirectory

Instead of pushing entire directories, create a clean public structure:

### Step 1: Create Public Branch

```bash
# Create a branch that will become public
git checkout -b public-export

# Remove private files
git rm -r data/ history/ .beads/ seed.md
git rm *.ogg *.mp3 *.wav 2>/dev/null || true

# Keep only public files
# Commit this state
git commit -m "Prepare for public export"
```

### Step 2: Push Public Branch

```bash
# Add public remote
git remote add public git@github.com:yourusername/hai-public.git

# Push public branch
git push public public-export:main
```

### Step 3: Use Subtree for Ongoing Sync

For ongoing updates, use subtree to push specific directories:

```bash
# After making changes to cmd/onboard/
git subtree push --prefix=cmd/onboard public main

# Or push entire cmd/ directory
git subtree push --prefix=cmd public main
```

## Recommended Workflow

### Option A: Manual Sync (Simplest)

1. **Work in private repo** as normal
2. **When ready to share updates:**
   ```bash
   # Push specific directory
   git subtree push --prefix=cmd public main
   ```

### Option B: Automated Sync Script

Create `scripts/sync-public.sh`:

```bash
#!/bin/bash
set -e

PUBLIC_REPO="git@github.com:yourusername/hai-public.git"
PUBLIC_BRANCH="main"

echo "Syncing to public repository..."

# Push cmd/ directory
echo "Pushing cmd/ directory..."
git subtree push --prefix=cmd $PUBLIC_REPO $PUBLIC_BRANCH

# Push other public directories if needed
# git subtree push --prefix=internal $PUBLIC_REPO $PUBLIC_BRANCH

echo "✅ Sync complete!"
```

### Option C: Separate Public Branch (Recommended)

1. **Create a `public` branch** in private repo
2. **Keep it clean** (no private files)
3. **Merge from main** when ready to share
4. **Push public branch** to public repo

```bash
# Create public branch
git checkout -b public
git rm -r data/ history/ .beads/  # Remove private files
git commit -m "Initial public export"

# Push to public repo
git remote add public git@github.com:yourusername/hai-public.git
git push public public:main

# When ready to update public:
git checkout public
git merge main --no-commit
# Manually resolve conflicts (remove private files)
git commit -m "Update public version"
git push public public:main
```

## What to Share vs Keep Private

### ✅ Share Publicly:
- `cmd/onboard/` - Onboarding server code
- `cmd/diarize/` - Diarization code
- `cmd/ingest/` - Audio ingestion code
- `internal/` - Shared packages (if created)
- `go.mod` - Go dependencies
- `README.md` - Public documentation
- `LICENSE` - License file
- `SETUP.md` - Setup instructions (sanitized)

### ❌ Keep Private:
- `data/` - Personal audio files and data
- `history/` - Planning and design documents
- `.beads/` - Issue tracking (personal)
- `seed.md` - Personal notes
- `.env` - API keys and secrets
- `*.ogg`, `*.mp3` - Audio files
- `lifelog.json` - Personal data

## Creating Public README

Create a public-facing README:

```markdown
# Hai - Audio Lifelog Processing

A system for processing and analyzing audio lifelogs using speaker diarization.

## Features

- Audio ingestion from Limitless API
- Speaker diarization using pyannote.audio
- Onboarding server for easy setup
- (Future: Elasticsearch storage, web interface)

## Setup

See [SETUP.md](SETUP.md) for installation instructions.

## License

MIT (or your chosen license)
```

## Best Practices

1. **Review before pushing**: Always review what you're pushing
2. **Use .gitignore**: Ensure private files are ignored
3. **Sanitize commits**: Remove any API keys or personal info from commit history
4. **Separate branches**: Use a `public` branch to keep things clean
5. **Document what's public**: Add a note in private repo about what's shared

## Alternative: Git Submodule (Not Recommended)

Git submodule is another option but has drawbacks:
- ❌ More complex to manage
- ❌ Requires users to clone submodules
- ❌ Harder to keep in sync

**Recommendation**: Stick with git subtree or separate branch approach.

## Quick Reference

```bash
# Initial setup
git remote add public git@github.com:yourusername/hai-public.git

# Push directory to public
git subtree push --prefix=cmd public main

# Pull updates from public (if others contribute)
git subtree pull --prefix=cmd public main --squash

# Check what would be pushed
git log public/main..HEAD -- cmd/
```

## Recommended Approach for Your Use Case

**Use Option C (Separate Public Branch)**:

1. Create `public` branch in private repo
2. Remove all private files from that branch
3. Push `public` branch to public repo
4. When ready to share updates:
   - Merge `main` into `public`
   - Remove any private files that got merged
   - Push `public` branch to public repo

This gives you:
- ✅ Clean separation
- ✅ Easy to review before sharing
- ✅ Can share selectively
- ✅ Simple workflow









