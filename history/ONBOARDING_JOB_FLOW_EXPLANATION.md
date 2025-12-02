# Onboarding "Job Submission" Explanation

## What is a "Job" in Onboarding Context?

A **job** is a single user request to process audio data. Here's the flow:

### User Flow

1. **User opens onboarding UI** (http://localhost:3000)
2. **User enters**:
   - Limitless API key
   - Date range (start date, end date)
3. **User clicks "Start Processing"** → This **submits a job**
4. **Onboarding server receives job** and starts processing:
   - Fetches audio from Limitless API
   - Runs diarization
   - Starts Docker services
   - Imports data to Elasticsearch
5. **User sees progress** and eventually gets redirected to full app

## Why "Job Submission"?

The term "job" is used because:
- It's an **asynchronous operation** (takes minutes/hours)
- User submits it and **waits for completion**
- Server tracks **job status** (pending, processing, completed, failed)
- Similar to job queues (though we're doing it synchronously in onboarding)

## Simplified: It's Just a Request

Think of it as:
- **User request**: "Process audio from Nov 22, 3pm-7pm"
- **Server response**: "OK, processing... here's your job ID: abc123"
- **User polls**: "What's the status of job abc123?"
- **Server responds**: "45% complete, diarizing..."

## Do Docker Services Receive Jobs?

**No!** The onboarding server handles everything:

```
User → Onboarding Server (port 3000)
         ↓
   1. Fetches audio (calls Limitless API)
   2. Runs diarization (calls Python subprocess)
   3. Starts Docker (exec.Command docker-compose)
   4. Waits for Docker services to be ready
   5. Imports data to Elasticsearch (via HTTP API)
         ↓
User → Full App (port 3001, served by Docker)
```

The Docker services (backend API, Elasticsearch) **don't receive jobs** - they just provide storage and querying. The onboarding server does all the work, then imports the results.

## Better Terminology

Instead of "job submission", we could call it:
- **"Process Request"** - User requests processing
- **"Import Request"** - User requests data import
- **"Setup Request"** - User requests setup/onboarding

But "job" is fine - it's a common pattern for async operations.

## In Code Terms

```go
// User submits a "job" (really just a request)
POST /api/submit
{
  "apiKey": "sk-...",
  "startDate": "2025-11-22T15:00:00Z",
  "endDate": "2025-11-22T19:00:00Z"
}

// Server responds with job ID
{
  "jobId": "abc123",
  "status": "processing"
}

// User polls for status
GET /api/status?jobId=abc123

// Server responds with progress
{
  "jobId": "abc123",
  "status": "diarizing",
  "progress": 45,
  "message": "Processing audio... 45% complete"
}
```

## Summary

- **"Job"** = User's request to process audio data
- **"Job submission"** = User clicking "Start Processing"
- **Docker services** = Just storage/querying, don't receive jobs
- **Onboarding server** = Does all the work, then imports to Docker services

It's really just a fancy way of saying "user request" with status tracking.









