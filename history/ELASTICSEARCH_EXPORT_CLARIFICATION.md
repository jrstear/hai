# Elasticsearch Export: Automatic vs Migration

## Current State: Automatic Export (Already Working)

The onboarding server **automatically** exports diarization results to Elasticsearch as part of the normal processing flow:

1. User submits job via web UI
2. Server downloads audio
3. Server runs diarization (or uses cached JSON)
4. **Server automatically exports to Elasticsearch** (if `ELASTICSEARCH_URL` is set)
5. Shows "Loading HH.json to Elasticsearch" in stdout

**Code location**: `onboard/internal/server/handlers.go` lines 630-671

This happens for:
- ✅ **New diarization jobs** (when user runs the onboarding server)
- ✅ **Cached diarization results** (if JSON exists, it still exports to ES)

## Tools Available

### 1. Onboarding Server (Automatic)
- **When**: During normal processing flow
- **What**: Automatically exports after diarization completes
- **Scope**: Only processes files requested in the current job
- **Use case**: Normal operation, new data

### 2. `load-es` CLI Tool (Manual, Single File)
- **When**: Manual invocation
- **What**: Loads a single diarization JSON file to Elasticsearch
- **Scope**: One file at a time
- **Use case**: Debugging, testing, loading a specific file
- **Location**: `onboard/cmd/load-es/main.go`

### 3. Migration Tool (hai-d8z) - To Be Built
- **When**: One-time migration of existing files
- **What**: Scans for ALL existing JSON files and migrates them
- **Scope**: All files in `data/` directory (batch operation)
- **Use case**: Migrating files created BEFORE Elasticsearch integration was added

## The Confusion

**Question**: When stdout says "Loading HH.json to Elasticsearch", is that the same as what d8z would do?

**Answer**: **Yes, it's the same operation** (both use `ExportResult()`), but different contexts:

- **Onboarding server**: Automatic, happens during normal flow for NEW jobs
- **Migration tool (d8z)**: One-time batch migration for EXISTING files

## Why Do We Need hai-d8z?

If you have existing JSON files that were created **before** the Elasticsearch integration was added, they won't be in Elasticsearch. The onboarding server only exports files it processes in NEW jobs.

**Example scenario**:
- You have 50 existing JSON files from November 2025
- You run the onboarding server for a NEW date range (December 2025)
- The server will export the December files automatically
- But the November files are still not in Elasticsearch
- **hai-d8z** would migrate all 50 November files in one batch

## Summary

| Tool | When | Scope | Purpose |
|------|------|-------|---------|
| **Onboarding Server** | Automatic, during job processing | Files in current job | Normal operation |
| **`load-es` CLI** | Manual, on-demand | Single file | Debugging/testing |
| **Migration Tool (d8z)** | One-time batch | All existing files | Migrate pre-ES files |

All three use the same underlying `ExportResult()` function - they just differ in **when** and **what** they process.













