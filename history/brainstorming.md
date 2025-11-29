# Project "Hai" - Brainstorming & Planning

## 1. Executive Summary
Building a personal Audio Lifelog Processing System ("Hai") using Limitless.ai data.
**Goal**: Enhance relationships and personal growth through AI-assisted analysis of conversations.
**Stack**: Go (Backend), Flutter (Frontend), Beads (PM), Antigravity (IDE).

## 2. Architecture Review
*   **Backend (Go)**: Excellent choice for performance, concurrency (processing audio), and portability (Mac/AWS Lambda).
*   **Frontend (Flutter)**: Good for single codebase across Web/iOS/Android.
    *   *Question*: Are you comfortable with Dart? (It's similar to TS/Java).
*   **Database**:
    *   **Local**: SQLite is perfect for the "personal experiment" phase.
    *   **Cloud**: RDS (Postgres) is a natural upgrade path.
    *   *Recommendation*: Use an ORM or query builder (like `sqlc` or `gorm`) that works well with both.

## 3. The "Diarization" Challenge
This is the critical technical hurdle. Limitless.ai gives "Unknown" speakers.
*   **Approach A**: Use a 3rd party API (Deepgram, Google STT) initially.
    *   *Pros*: High quality, easy to start.
    *   *Cons*: Cost, privacy (sending audio out).
*   **Approach B**: Local/Cloud GPU with Pyannote (or similar).
    *   *Pros*: Privacy, control, cheaper at scale (if self-hosted).
    *   *Cons*: Complex setup, hardware requirements (M1 is good, but might be slow for bulk processing).
*   **Speaker Identification**: Diarization separates speakers (Speaker A, Speaker B). *Identification* (mapping Speaker A -> "Mom") requires a vector DB or embedding comparison.
    *   *Plan*: We need to store "voice embeddings" for each contact.

## 4. Data Model (The "Seed" Concept)
Refining the "Hierarchical Grouping" from `seed.jpg` (as described).
*   **Event**: Base unit.
    *   **Type**: `Recording` (raw audio), `Conversation` (logical grouping), `Meeting` (calendar event).
    *   **Attributes**: Start, End, Participants (List of IDs).
*   **Contact**:
    *   `ID`, `Name`, `VoiceEmbedding` (vector), `RelationshipStats`.

## 5. Beads Integration
We will use `bd` to track this.
*   **Epics**:
    1.  `Data Ingestion` (Limitless API -> Local Storage)
    2.  `Diarization Pipeline` (Audio -> Segments -> Embeddings)
    3.  `Contact Management` (UI for tagging)
    4.  `Visualization` (Timeline/Calendar)
    5.  `PRM/Insights` (The "AI Coach")

## 6. Immediate Next Steps (Proposed)
1.  **Define Schema**: Create a SQL schema for `Recordings`, `Speakers`, `Contacts`.
2.  **Prototype Ingestion**: Write a Go script to fetch data from Limitless.
3.  **Feasibility Study**: Test a diarization library on `audio.ogg`.

---
**User Feedback Needed**:
1.  Does this structure align with your vision?
2.  Preference on Diarization (Cloud API vs Local Lib)?
3.  Any specific constraints on the "Personal Landscape" (e.g., budget for APIs)?
