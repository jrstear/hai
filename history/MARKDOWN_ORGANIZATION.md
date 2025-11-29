# Markdown File Organization

## Structure

### Root Level (Current/Active Documentation)
- `AGENTS.md` - AI agent instructions for beads workflow
- `seed.md` - Project vision and requirements
- `SETUP.md` - Development environment setup guide

### cmd/diarize/ (Tool-Specific Documentation)
- `test_case_420_480s.md` - Active test case documentation for multi-speaker segment

### history/ (Planning & Archived Results)
- **Planning Documents:**
  - `SPEAKER_DATABASE_SCHEMA.md` - Current database schema design
  - `SCHEMA_DISCUSSION_SUMMARY.md` - Schema discussion summary
  - `REMAINING_CONSIDERATIONS.md` - Implementation considerations
  - `CONTACTS_INTEGRATION_NOTES.md` - Contacts integration planning
  - `BYTE_OFFSET_EFFICIENCY_CHECK.md` - Byte offset analysis
  - `IMPLEMENTATION_READY_SUMMARY.md` - Implementation readiness summary
  - `brainstorming.md` - Early project brainstorming
  
- **Archived/Outdated:**
  - `SPEAKER_EMBEDDINGS_STORAGE.md` - Superseded by SPEAKER_DATABASE_SCHEMA.md
  - `diarization_results_summary.md` - Results from specific 15.ogg run (Nov 28, 2025)

## Guidelines

- **Root level**: Active documentation used regularly (setup, agent instructions, project vision)
- **cmd/diarize/**: Active tool-specific documentation (test cases, usage references)
- **history/**: Planning documents, discussions, and archived results

When creating new planning/design documents, add them to `history/` to keep the repo root clean.

