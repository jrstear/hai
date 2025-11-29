#!/usr/bin/env python3
"""
Diarize an audio file and save results to JSON alongside the audio file.

Results are saved as <audio_file_basename>.json in the same directory as the audio file.
If results already exist, they are reused unless --force is specified.
"""

import time
import os
import sys
import json
import argparse
from pathlib import Path

import torch
import numpy as np
import soundfile as sf

# Monkey-patch torch.load to allow loading pyannote models
_original_torch_load = torch.load
def _patched_torch_load(*args, **kwargs):
    kwargs['weights_only'] = False
    return _original_torch_load(*args, **kwargs)
torch.load = _patched_torch_load

from pyannote.audio import Pipeline


def get_result_path(audio_file: str) -> str:
    """Get the path where diarization results should be saved."""
    audio_path = Path(audio_file)
    result_path = audio_path.parent / f"{audio_path.stem}.json"
    return str(result_path)


def load_cached_results(result_file: str) -> dict:
    """Load cached diarization results from JSON file."""
    try:
        with open(result_file, 'r') as f:
            return json.load(f)
    except Exception as e:
        return None


def save_results(result_file: str, results: dict):
    """Save diarization results to JSON file."""
    result_path = Path(result_file)
    result_path.parent.mkdir(parents=True, exist_ok=True)
    
    with open(result_file, 'w') as f:
        json.dump(results, f, indent=2)
    print(f"Results saved to: {result_file}")


def load_audio(audio_file: str):
    """Load audio file, handling OGG format issues."""
    try:
        waveform, sample_rate = sf.read(audio_file)
    except Exception as e:
        # Try converting with ffmpeg first
        import subprocess
        temp_wav = "/tmp/temp_audio_diarize.wav"
        cmd = ['ffmpeg', '-i', audio_file, '-y', temp_wav]
        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode == 0:
            waveform, sample_rate = sf.read(temp_wav)
            os.remove(temp_wav)
        else:
            raise Exception(f"Failed to load audio: {e}")
    
    if waveform.ndim == 1:
        waveform = waveform[np.newaxis, :]
        audio_duration = len(waveform[0]) / sample_rate
    else:
        waveform = waveform.T
        audio_duration = waveform.shape[1] / sample_rate
    
    waveform_tensor = torch.from_numpy(waveform.astype(np.float32))
    audio = {
        "waveform": waveform_tensor,
        "sample_rate": sample_rate
    }
    
    return audio, audio_duration


def run_diarization(audio_file: str, hf_token: str, force: bool = False) -> dict:
    """Run diarization on an audio file, using cache if available."""
    result_file = get_result_path(audio_file)
    
    # Check for cached results
    if not force and os.path.exists(result_file):
        cached = load_cached_results(result_file)
        if cached:
            print(f"Using cached results from: {result_file}")
            print(f"  Cached on: {cached.get('timestamp', 'unknown')}")
            print(f"  Speakers: {cached.get('speaker_count', 'unknown')}")
            print(f"  Segments: {cached.get('segment_count', 'unknown')}")
            return cached
    
    print(f"Diarizing: {audio_file}")
    
    # Load pipeline
    print("Loading pipeline...")
    start_load = time.time()
    pipeline = Pipeline.from_pretrained(
        "pyannote/speaker-diarization-3.1",
        token=hf_token
    )
    
    # Move to GPU if available
    device = "cpu"
    if torch.backends.mps.is_available():
        print("Using MPS acceleration")
        pipeline.to(torch.device("mps"))
        device = "mps"
    elif torch.cuda.is_available():
        print("Using CUDA acceleration")
        pipeline.to(torch.device("cuda"))
        device = "cuda"
    
    load_time = time.time() - start_load
    print(f"Pipeline loaded in {load_time:.2f} seconds")
    
    # Load audio
    print("Loading audio...")
    audio, audio_duration = load_audio(audio_file)
    
    # Run diarization
    print("Running diarization...")
    start_run = time.time()
    diarization = pipeline(audio)
    run_time = time.time() - start_run
    
    # Extract results
    annotation = diarization.speaker_diarization
    speakers = sorted(set(annotation.labels()))
    
    segments = []
    for segment, _, label in annotation.itertracks(yield_label=True):
        segments.append({
            'speaker': label,
            'start': float(segment.start),
            'end': float(segment.end),
            'duration': float(segment.end - segment.start)
        })
    
    # Extract speaker embeddings
    speaker_embeddings = {}
    if hasattr(diarization, 'speaker_embeddings') and diarization.speaker_embeddings is not None:
        # speaker_embeddings is a numpy array with shape (num_speakers, embedding_dim)
        embeddings_array = diarization.speaker_embeddings
        
        # Convert numpy array to list for JSON serialization
        if isinstance(embeddings_array, np.ndarray):
            embeddings_list = embeddings_array.tolist()
            
            # Map each speaker to its embedding
            # The order should match the speakers list, but we'll verify
            for idx, speaker in enumerate(speakers):
                if idx < len(embeddings_list):
                    # Store as list of floats for JSON serialization
                    speaker_embeddings[speaker] = [float(x) for x in embeddings_list[idx]]
        
        print(f"Extracted embeddings for {len(speaker_embeddings)} speakers")
        if speaker_embeddings:
            # Show embedding dimension
            first_speaker = list(speaker_embeddings.keys())[0]
            embedding_dim = len(speaker_embeddings[first_speaker])
            print(f"  Embedding dimension: {embedding_dim}")
    
    # Build results dictionary
    results = {
        'audio_file': os.path.abspath(audio_file),
        'timestamp': time.strftime('%Y-%m-%d %H:%M:%S'),
        'audio_duration': float(audio_duration),
        'processing_time': float(run_time),
        'rtf': float(run_time / audio_duration),
        'device': device,
        'speaker_count': len(speakers),
        'speakers': speakers,
        'segment_count': len(segments),
        'segments': segments,
        'speaker_embeddings': speaker_embeddings if speaker_embeddings else None
    }
    
    print(f"Diarization completed in {run_time:.2f} seconds")
    print(f"Audio duration: {audio_duration:.2f} seconds")
    print(f"RTF: {run_time/audio_duration:.3f}x")
    print(f"Detected {len(speakers)} speaker(s): {speakers}")
    print(f"Found {len(segments)} segments")
    
    # Save results
    save_results(result_file, results)
    
    return results


def main():
    parser = argparse.ArgumentParser(
        description='Diarize an audio file and save results to JSON'
    )
    parser.add_argument('audio_file', help='Path to audio file to diarize')
    parser.add_argument('--force', '-f', action='store_true',
                       help='Force re-diarization even if cached results exist')
    parser.add_argument('--output', '-o',
                       help='Output JSON file path (default: <audio_file>.json)')
    
    args = parser.parse_args()
    
    # Check for HF token
    hf_token = os.environ.get("HF_TOKEN")
    if not hf_token:
        print("Error: HF_TOKEN environment variable not set", file=sys.stderr)
        sys.exit(1)
    
    # Check if audio file exists
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}", file=sys.stderr)
        sys.exit(1)
    
    # Run diarization
    results = run_diarization(args.audio_file, hf_token, force=args.force)
    
    # Output to custom path if specified
    if args.output:
        save_results(args.output, results)


if __name__ == "__main__":
    main()

