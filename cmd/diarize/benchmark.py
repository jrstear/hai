import time
import os
import torch
import sys
import numpy as np

# Monkey-patch torch.load to allow loading pyannote models
# We trust HuggingFace models
_original_torch_load = torch.load
def _patched_torch_load(*args, **kwargs):
    kwargs['weights_only'] = False
    return _original_torch_load(*args, **kwargs)
torch.load = _patched_torch_load

from pyannote.audio import Pipeline

def benchmark(audio_file):
    print(f"Benchmarking diarization on {audio_file}...")
    
    # Check for HF token
    hf_token = os.environ.get("HF_TOKEN")
    if not hf_token:
        print("Error: HF_TOKEN environment variable not set. Please set it to your Hugging Face access token.")
        print("You need to accept the user conditions on https://hf.co/pyannote/speaker-diarization-3.1")
        sys.exit(1)

    # Load pipeline
    print("Loading pipeline...")
    start_load = time.time()
    try:
        pipeline = Pipeline.from_pretrained(
            "pyannote/speaker-diarization-3.1",
            token=hf_token
        )
    except Exception as e:
        print(f"Failed to load pipeline: {e}")
        sys.exit(1)
        
    # Move to GPU if available (MPS for Mac M1)
    if torch.backends.mps.is_available():
        print("Using MPS (Metal Performance Shaders) acceleration")
        pipeline.to(torch.device("mps"))
    elif torch.cuda.is_available():
        print("Using CUDA acceleration")
        pipeline.to(torch.device("cuda"))
    else:
        print("Using CPU (Warning: will be slow)")

    end_load = time.time()
    print(f"Pipeline loaded in {end_load - start_load:.2f} seconds")

    # Run diarization
    print("Running diarization...")
    start_run = time.time()
    
    # Import here to avoid issues if not available
    import soundfile as sf
    
    try:
        # Load audio explicitly using soundfile to avoid AudioDecoder issues
        waveform, sample_rate = sf.read(audio_file)
        
        # Calculate audio duration for RTF before processing
        if waveform.ndim == 1:
            audio_duration = len(waveform) / sample_rate
        else:
            audio_duration = waveform.shape[0] / sample_rate
        
        # Convert to torch tensor and ensure correct shape (channel, time)
        if waveform.ndim == 1:
            waveform = waveform[np.newaxis, :]  # Add channel dimension
        else:
            waveform = waveform.T  # Transpose to (channel, time)
        
        waveform_tensor = torch.from_numpy(waveform.astype(np.float32))
        
        # Create audio dict that pyannote expects
        audio = {
            "waveform": waveform_tensor,
            "sample_rate": sample_rate
        }
        
        diarization = pipeline(audio)
    except Exception as e:
        print(f"Diarization failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
        
    end_run = time.time()
    duration = end_run - start_run
    
    print(f"Diarization completed in {duration:.2f} seconds")
    print(f"Audio duration: {audio_duration:.2f} seconds")
    print(f"Real-Time Factor (RTF): {duration/audio_duration:.3f}x")
    
    print("\nResults:")
    try:
        # Access the speaker_diarization annotation from DiarizeOutput
        annotation = diarization.speaker_diarization
        
        # Get list of unique speakers using labels() method
        speakers = sorted(set(annotation.labels()))
        print(f"Detected {len(speakers)} speaker(s): {speakers}")
        
        # Count segments per speaker
        speaker_segments = {}
        speaker_duration = {}
        for segment, _, label in annotation.itertracks(yield_label=True):
            if label not in speaker_segments:
                speaker_segments[label] = 0
                speaker_duration[label] = 0.0
            speaker_segments[label] += 1
            speaker_duration[label] += segment.end - segment.start
        
        print(f"\nSpeaker statistics:")
        for speaker in speakers:
            print(f"  {speaker}: {speaker_segments[speaker]} segments, {speaker_duration[speaker]:.1f}s total ({speaker_duration[speaker]/audio_duration*100:.1f}%)")
        
        print(f"\nTimeline:")
        # Iterate over the annotation and print segments
        for segment, _, label in annotation.itertracks(yield_label=True):
            print(f"  [{segment.start:.1f}s - {segment.end:.1f}s] {label}")
            
    except AttributeError as e:
        print(f"Error accessing diarization results: {e}")
        print(f"Diarization type: {type(diarization)}")
        print(f"Diarization attributes: {dir(diarization)}")
        if hasattr(diarization, 'speaker_diarization'):
            print(f"speaker_diarization type: {type(diarization.speaker_diarization)}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python benchmark.py <audio_file>")
        sys.exit(1)
        
    audio_file = sys.argv[1]
    benchmark(audio_file)
