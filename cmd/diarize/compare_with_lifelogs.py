#!/usr/bin/env python3
"""
Compare pyannote diarization results with Limitless API lifelog data.
Finds multi-speaker sections in lifelogs and compares with local diarization.
"""

import json
import os
import sys
import time
from datetime import datetime, timedelta
from collections import defaultdict
from typing import Dict, List, Tuple

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


def load_lifelogs(lifelog_file: str) -> List[Dict]:
    """Load lifelogs from JSON file."""
    with open(lifelog_file, 'r') as f:
        data = json.load(f)
    # Handle both list format and wrapped format
    if isinstance(data, list):
        return data
    return data.get('data', {}).get('lifelogs', [])


def find_multi_speaker_sections(lifelogs: List[Dict], min_speakers: int = 2) -> List[Dict]:
    """Find lifelog sections with multiple speakers."""
    multi_speaker = []
    
    for lifelog in lifelogs:
        # Count unique speakers in this lifelog
        speakers = set()
        speaker_segments = []
        
        for content in lifelog.get('contents', []):
            if content.get('type') == 'blockquote' and content.get('speakerName'):
                speaker_name = content['speakerName']
                speakers.add(speaker_name)
                
                if 'startOffsetMs' in content and 'endOffsetMs' in content:
                    speaker_segments.append({
                        'speaker': speaker_name,
                        'start_ms': content['startOffsetMs'],
                        'end_ms': content['endOffsetMs'],
                        'content': content.get('content', '')
                    })
        
        if len(speakers) >= min_speakers and len(speaker_segments) > 0:
            multi_speaker.append({
                'lifelog': lifelog,
                'speakers': list(speakers),
                'speaker_count': len(speakers),
                'segments': speaker_segments,
                'start_time': lifelog.get('startTime'),
                'end_time': lifelog.get('endTime'),
                'duration_ms': sum(s['end_ms'] - s['start_ms'] for s in speaker_segments)
            })
    
    # Sort by speaker count (descending) and duration (descending)
    multi_speaker.sort(key=lambda x: (x['speaker_count'], x['duration_ms']), reverse=True)
    return multi_speaker


def parse_lifelog_time(time_str: str) -> datetime:
    """Parse ISO 8601 time string from lifelog."""
    # Handle both with and without timezone
    if time_str.endswith('Z'):
        time_str = time_str[:-1] + '+00:00'
    return datetime.fromisoformat(time_str.replace('Z', '+00:00'))


def find_audio_file_for_lifelog(lifelog_start: str, base_dir: str = "data/audio") -> str:
    """Find the audio file that contains this lifelog."""
    lifelog_time = parse_lifelog_time(lifelog_start)
    
    # Audio files are organized as YYYY/MM/DD/HH.ogg
    # Each file contains one hour of audio starting at HH:00
    audio_path = os.path.join(
        base_dir,
        lifelog_time.strftime("%Y"),
        lifelog_time.strftime("%m"),
        lifelog_time.strftime("%d"),
        f"{lifelog_time.strftime('%H')}.ogg"
    )
    
    return audio_path


def extract_segment_from_audio(audio_file: str, start_ms: int, end_ms: int, output_file: str):
    """Extract a segment from an audio file."""
    import subprocess
    
    # Use ffmpeg to extract segment
    start_sec = start_ms / 1000.0
    duration = (end_ms - start_ms) / 1000.0
    
    cmd = [
        'ffmpeg', '-i', audio_file,
        '-ss', str(start_sec),
        '-t', str(duration),
        '-acodec', 'copy',
        '-y',  # Overwrite output
        output_file
    ]
    
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        # Fallback: use soundfile and numpy
        waveform, sample_rate = sf.read(audio_file)
        start_sample = int(start_sec * sample_rate)
        end_sample = int((start_sec + duration) * sample_rate)
        segment = waveform[start_sample:end_sample]
        sf.write(output_file, segment, sample_rate)
    else:
        print(f"Extracted segment using ffmpeg: {output_file}")


def load_diarization_results(result_file: str) -> Dict:
    """Load diarization results from cached JSON file."""
    try:
        with open(result_file, 'r') as f:
            data = json.load(f)
        # Convert to expected format
        return {
            'speakers': data.get('speakers', []),
            'speaker_count': data.get('speaker_count', 0),
            'segments': data.get('segments', []),
            'audio_duration': data.get('audio_duration', 0),
            'processing_time': data.get('processing_time', 0),
            'rtf': data.get('rtf', 0)
        }
    except Exception as e:
        return None


def get_result_path(audio_file: str) -> str:
    """Get the path where diarization results should be saved."""
    import os
    from pathlib import Path
    audio_path = Path(audio_file)
    result_path = audio_path.parent / f"{audio_path.stem}.json"
    return str(result_path)


def run_diarization(audio_file: str, hf_token: str, force: bool = False) -> Dict:
    """Run diarization on an audio file, using cache if available."""
    result_file = get_result_path(audio_file)
    
    # Check for cached results
    if not force and os.path.exists(result_file):
        cached = load_diarization_results(result_file)
        if cached:
            print(f"\n{'='*60}")
            print(f"Using cached diarization results from: {result_file}")
            print(f"{'='*60}")
            return cached
    
    # Need to run diarization - use the diarize.py script
    print(f"\n{'='*60}")
    print(f"Running diarization on: {audio_file}")
    print(f"{'='*60}")
    
    # Import diarize module functions
    import subprocess
    import sys
    
    # Call diarize.py script
    script_path = os.path.join(os.path.dirname(__file__), 'diarize.py')
    cmd = [sys.executable, script_path]
    if force:
        cmd.append('--force')
    cmd.append(audio_file)
    
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"Error running diarization: {result.stderr}", file=sys.stderr)
        raise Exception(f"Diarization failed: {result.stderr}")
    
    # Load the results that were just saved
    cached = load_diarization_results(result_file)
    if not cached:
        raise Exception("Diarization completed but results not found")
    
    return cached


def compare_results(lifelog_data: Dict, diarization_data: Dict) -> Dict:
    """Compare lifelog speaker data with diarization results."""
    lifelog_speakers = set(lifelog_data['speakers'])
    diarization_speakers = set(diarization_data['speakers'])
    
    # Note: Lifelog uses names like "Unknown", "Jon Stearley", etc.
    # Diarization uses IDs like "SPEAKER_00", "SPEAKER_01", etc.
    # We can't directly match names, but we can compare counts and timing
    
    comparison = {
        'lifelog_speaker_count': len(lifelog_speakers),
        'lifelog_speakers': list(lifelog_speakers),
        'diarization_speaker_count': len(diarization_speakers),
        'diarization_speakers': list(diarization_speakers),
        'speaker_count_match': len(lifelog_speakers) == len(diarization_speakers),
        'lifelog_segments': len(lifelog_data['segments']),
        'diarization_segments': len(diarization_data['segments'])
    }
    
    return comparison


def print_comparison_report(lifelog_info: Dict, diarization_data: Dict, comparison: Dict):
    """Print a detailed comparison report."""
    print(f"\n{'='*60}")
    print("COMPARISON REPORT")
    print(f"{'='*60}")
    
    print(f"\nLifelog Information:")
    print(f"  Title: {lifelog_info['lifelog'].get('title', 'N/A')}")
    print(f"  Time: {lifelog_info['start_time']} to {lifelog_info['end_time']}")
    print(f"  Speakers (Limitless API): {comparison['lifelog_speaker_count']}")
    for speaker in comparison['lifelog_speakers']:
        print(f"    - {speaker}")
    print(f"  Segments: {comparison['lifelog_segments']}")
    
    print(f"\nDiarization Results (Local):")
    print(f"  Speakers detected: {comparison['diarization_speaker_count']}")
    for speaker in comparison['diarization_speakers']:
        print(f"    - {speaker}")
    print(f"  Segments: {comparison['diarization_segments']}")
    print(f"  Processing time: {diarization_data['processing_time']:.2f}s")
    print(f"  RTF: {diarization_data['rtf']:.3f}x")
    
    print(f"\nComparison:")
    if comparison['speaker_count_match']:
        print(f"  ✓ Speaker count matches!")
    else:
        print(f"  ✗ Speaker count mismatch: {comparison['lifelog_speaker_count']} (API) vs {comparison['diarization_speaker_count']} (local)")
    
    print(f"\nSpeaker Timeline (Limitless API):")
    for seg in lifelog_info['segments'][:10]:  # Show first 10
        print(f"  [{seg['start_ms']/1000:.1f}s - {seg['end_ms']/1000:.1f}s] {seg['speaker']}: {seg['content'][:50]}")
    if len(lifelog_info['segments']) > 10:
        print(f"  ... and {len(lifelog_info['segments']) - 10} more segments")
    
    print(f"\nSpeaker Timeline (Local Diarization):")
    for seg in diarization_data['segments'][:10]:  # Show first 10
        print(f"  [{seg['start']:.1f}s - {seg['end']:.1f}s] {seg['speaker']}")
    if len(diarization_data['segments']) > 10:
        print(f"  ... and {len(diarization_data['segments']) - 10} more segments")


def main():
    import argparse
    
    parser = argparse.ArgumentParser(
        description='Compare pyannote diarization results with Limitless API lifelog data'
    )
    parser.add_argument('lifelog_file', help='Path to lifelog JSON file')
    parser.add_argument('audio_file', nargs='?', help='Path to audio file (optional, will find automatically)')
    parser.add_argument('--rediarize', '-r', action='store_true',
                       help='Force re-diarization even if cached results exist')
    
    args = parser.parse_args()
    
    lifelog_file = args.lifelog_file
    
    # Check for HF token
    hf_token = os.environ.get("HF_TOKEN")
    if not hf_token:
        print("Error: HF_TOKEN environment variable not set")
        sys.exit(1)
    
    # Load lifelogs
    print(f"Loading lifelogs from {lifelog_file}...")
    lifelogs = load_lifelogs(lifelog_file)
    print(f"Loaded {len(lifelogs)} lifelogs")
    
    # Find multi-speaker sections
    print("\nFinding multi-speaker sections...")
    multi_speaker = find_multi_speaker_sections(lifelogs, min_speakers=2)
    print(f"Found {len(multi_speaker)} multi-speaker lifelogs")
    
    if len(multi_speaker) == 0:
        print("No multi-speaker sections found!")
        sys.exit(1)
    
    # Show top candidates
    print("\nTop multi-speaker candidates:")
    for i, section in enumerate(multi_speaker[:5], 1):
        print(f"{i}. {section['speaker_count']} speakers, "
              f"{section['duration_ms']/1000:.1f}s, "
              f"{section['lifelog'].get('title', 'N/A')[:50]}")
    
    # Process the best candidate
    best = multi_speaker[0]
    print(f"\n{'='*60}")
    print(f"Processing best candidate:")
    print(f"  {best['lifelog'].get('title', 'N/A')}")
    print(f"  {best['speaker_count']} speakers, {best['duration_ms']/1000:.1f}s duration")
    print(f"{'='*60}")
    
    # Find corresponding audio file
    audio_file = args.audio_file
    else:
        # Try to find a multi-speaker section that matches available audio
        available_audio_dir = "data/audio/2025/11/22"
        available_files = []
        if os.path.exists(available_audio_dir):
            for f in sorted(os.listdir(available_audio_dir)):
                if f.endswith('.ogg'):
                    available_files.append(os.path.join(available_audio_dir, f))
        
        if available_files:
            print(f"\nAvailable audio files: {available_files}")
            # Find a lifelog that matches one of these files
            best_match = None
            best_section = None
            
            for section in multi_speaker:
                audio_file_candidate = find_audio_file_for_lifelog(section['start_time'])
                if audio_file_candidate in available_files:
                    best_match = audio_file_candidate
                    best_section = section
                    break
            
            if best_match:
                audio_file = best_match
                best = best_section
                print(f"\nUsing matching audio file: {audio_file}")
                print(f"Lifelog: {best['lifelog'].get('title', 'N/A')}")
            else:
                # Just use the first available audio file and best multi-speaker section
                audio_file = available_files[0]
                print(f"\nUsing first available audio file: {audio_file}")
                print("Note: This may not exactly match the lifelog time")
        else:
            audio_file = find_audio_file_for_lifelog(best['start_time'])
            if not os.path.exists(audio_file):
                print(f"\nError: Audio file not found: {audio_file}")
                print("Please specify audio file as second argument")
                sys.exit(1)
    
    if not os.path.exists(audio_file):
        print(f"Error: Audio file not found: {audio_file}")
        sys.exit(1)
    
    # Calculate offset in audio file (if lifelog matches)
    audio_to_diarize = audio_file
    segment_file = None
    
    try:
        lifelog_start = parse_lifelog_time(best['start_time'])
        # Extract hour from audio filename
        audio_hour = int(os.path.basename(audio_file).replace('.ogg', ''))
        audio_start = lifelog_start.replace(hour=audio_hour, minute=0, second=0, microsecond=0)
        offset_ms = int((lifelog_start - audio_start).total_seconds() * 1000)
        
        if offset_ms >= 0:
            print(f"\nLifelog starts at offset: {offset_ms}ms ({offset_ms/1000:.1f}s) in audio file")
            
            # Extract segment (if duration is reasonable and offset is valid)
            duration_ms = best['duration_ms']
            if 0 < duration_ms < 600000 and offset_ms >= 0:  # Less than 10 minutes
                print(f"\nExtracting segment ({duration_ms/1000:.1f}s)...")
                segment_file = "/tmp/diarize_segment.ogg"
                try:
                    extract_segment_from_audio(audio_file, offset_ms, offset_ms + duration_ms, segment_file)
                    if os.path.exists(segment_file) and os.path.getsize(segment_file) > 0:
                        audio_to_diarize = segment_file
                    else:
                        print("Extracted segment appears empty, using full file")
                        audio_to_diarize = audio_file
                except Exception as e:
                    print(f"Failed to extract segment: {e}")
                    print("Using full audio file instead")
                    audio_to_diarize = audio_file
            else:
                print(f"Using full audio file (segment duration: {duration_ms/1000:.1f}s)")
        else:
            print(f"\nLifelog time doesn't match audio file hour, using full audio file")
    except Exception as e:
        print(f"\nCould not calculate offset: {e}, using full audio file")
    
    # Run diarization (will use cache unless --rediarize is set)
    diarization_data = run_diarization(audio_to_diarize, hf_token, force=args.rediarize)
    
    # Compare results
    comparison = compare_results(best, diarization_data)
    
    # Print report
    print_comparison_report(best, diarization_data, comparison)
    
    # Cleanup
    if segment_file and os.path.exists(segment_file) and segment_file != audio_file:
        os.remove(segment_file)


if __name__ == "__main__":
    main()

