# End of Session Checklist

## Pre-Commit Checklist

- [ ] ✅ All sensitive files ignored (`.env` checked)
- [ ] ✅ Planning docs organized in `history/`
- [ ] ✅ Code saved and linted
- [ ] ✅ Documentation references added to beads issues
- [ ] ✅ Beads issues synced (or will auto-sync on commit)

## What to Commit Today

### Planning & Documentation
- [x] `history/*.md` - All planning documents (12 files)
- [x] `AGENTS.md` - Updated with beads workflow
- [x] `.gitignore` - Created to protect secrets
- [x] `history/BEADS_DOCUMENTATION_REFERENCE.md` - New practice doc

### Code & Tools
- [x] `cmd/diarize/*.py` - Diarization tools
- [x] `cmd/diarize/test_case_420_480s.md` - Test case reference

### Configuration
- [x] `.gitattributes` - Beads merge strategy
- [x] `.beads/` - Beads configuration (but not .db files)

## Commit Message Template

```
Schema design and planning documentation

- Created comprehensive database schema design
- Organized planning docs into history/ directory
- Added documentation references to all beads issues
- Established cleanup and documentation practices

Planning documents:
- SPEAKER_DATABASE_SCHEMA.md - Complete schema design
- REMAINING_CONSIDERATIONS.md - All decisions documented
- CONTACTS_INTEGRATION_NOTES.md - Contacts workflow
- Plus 9 additional planning documents

Beads issues:
- All 14 open issues now reference relevant documentation
- Ready for implementation phase

References: hai-tw5, hai-645, hai-d8z, hai-eog
```

## Post-Commit Verification

- [ ] `git status` shows clean working directory
- [ ] `.env` is NOT in git status
- [ ] Beads issues are committed (check `.beads/issues.jsonl`)
- [ ] All planning docs are in `history/`

## Next Session

1. Review: `bd ready --json` to see what's ready to work on
2. Start with: `hai-tw5` (database schema implementation)
3. Reference: `history/IMPLEMENTATION_READY_SUMMARY.md` for full checklist

