-- Initial schema for speakers, recordings, and segments
-- Supports SQLite with WAL mode for better concurrency

-- Enable WAL mode for better concurrency
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- Global speaker registry
-- Each speaker has a unique ID and embedding vector
CREATE TABLE IF NOT EXISTS speakers (
  id TEXT PRIMARY KEY,                    -- e.g., 'spkr_abc123' (UUID-based)
  embedding BLOB NOT NULL,                -- 256 floats as binary (1024 bytes)
  first_seen TIMESTAMP NOT NULL,          -- When this speaker was first detected
  last_seen TIMESTAMP NOT NULL,           -- When this speaker was last detected
  contact_id TEXT,                        -- NULL until associated with contact
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE SET NULL
);

-- Audio recordings metadata
-- Each recording represents a single audio file (typically 1-hour chunk)
CREATE TABLE IF NOT EXISTS recordings (
  id TEXT PRIMARY KEY,                    -- e.g., 'rec_2025_11_22_15'
  file_path TEXT NOT NULL UNIQUE,         -- 'data/2025/11/22/15.ogg' (relative to data root)
  start_time TIMESTAMP NOT NULL,          -- When recording started (UTC)
  duration_seconds REAL NOT NULL,         -- Total duration in seconds
  sample_rate INTEGER,                    -- Audio sample rate (optional)
  format TEXT,                            -- 'ogg', 'mp3', etc. (optional)
  diarized_at TIMESTAMP,                  -- When diarization completed (optional)
  processing_time REAL,                   -- Diarization processing time in seconds (optional)
  rtf REAL,                               -- Real-time factor (processing_time / duration) (optional)
  device TEXT,                            -- Device used: 'mps', 'cpu', 'cuda' (optional)
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Speaker segments (normalized, NOT JSON)
-- Each segment represents a time period during which a single speaker speaks
-- Equivalent to Limitless API's "blockquote" type
CREATE TABLE IF NOT EXISTS segments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  speaker_id TEXT NOT NULL,               -- Global speaker ID (references speakers.id)
  recording_id TEXT NOT NULL,             -- Which recording (references recordings.id)
  local_speaker_id TEXT,                  -- Original SPEAKER_XX from diarization (optional, for debugging)
  
  -- Time-based references (always present)
  start_time REAL NOT NULL,               -- Start in seconds (float, relative to recording start)
  end_time REAL NOT NULL,                 -- End in seconds (float, relative to recording start)
  duration REAL NOT NULL,                 -- Duration in seconds (end_time - start_time)
  
  -- Byte-based references (for fast seeking, DASH-style HTTP Range requests)
  start_byte_offset INTEGER,              -- NULL if not indexed yet
  end_byte_offset INTEGER,                -- NULL if not indexed yet
  
  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (speaker_id) REFERENCES speakers(id) ON DELETE CASCADE,
  FOREIGN KEY (recording_id) REFERENCES recordings(id) ON DELETE CASCADE,
  
  -- Constraints
  CHECK (end_time > start_time),
  CHECK (duration > 0),
  CHECK (start_byte_offset IS NULL OR start_byte_offset >= 0),
  CHECK (end_byte_offset IS NULL OR end_byte_offset >= start_byte_offset)
);

-- Indexes for common queries

-- Find all segments for a speaker
CREATE INDEX IF NOT EXISTS idx_speaker_segments ON segments(speaker_id, start_time);

-- Find all segments in a recording
CREATE INDEX IF NOT EXISTS idx_recording_segments ON segments(recording_id, start_time, end_time);

-- Time range queries (e.g., "segments between 10:00 and 11:00")
CREATE INDEX IF NOT EXISTS idx_time_range ON segments(recording_id, start_time, end_time);

-- Byte range queries (for HTTP Range requests)
CREATE INDEX IF NOT EXISTS idx_byte_range ON segments(recording_id, start_byte_offset, end_byte_offset);

-- Find recordings by time
CREATE INDEX IF NOT EXISTS idx_recordings_time ON recordings(start_time);

-- Find speakers by contact
CREATE INDEX IF NOT EXISTS idx_speakers_contact ON speakers(contact_id) WHERE contact_id IS NOT NULL;

-- Find speakers by last seen (for cleanup/archival)
CREATE INDEX IF NOT EXISTS idx_speakers_last_seen ON speakers(last_seen);









