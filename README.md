Video Compilation Tool Enhancements
I have enhanced the 
video_processor.py
 tool to meet your requirements.

Changes Implemented
1. Segment Count Range
Old Behavior: Fixed number of segments per video.
New Behavior: You can now specify a Min and Max number of segments per video (e.g., 3 to 4). The tool will randomly choose a count within this range for each video.
2. Randomization Options
Randomize Segment Selection (Time):
Enabled (Default): Segments are chosen randomly from the available duration (after 5 minutes).
Disabled: Segments are chosen sequentially starting from the 5-minute mark.
Shuffle Output Segments (Video Order):
Enabled: All extracted segments from all videos are shuffled together.
Disabled: Segments appear in the order of the input videos (Video 1 segments, then Video 2 segments, etc.).
3. GPU Acceleration
The tool already supported GPU acceleration via the "Enable GPU Acceleration" checkbox.
It uses h264_nvenc for encoding when enabled.
I verified this logic is preserved and correctly integrated.
How to Use
Run the script: python video_processor.py
Select multiple video files.
Set Segments per video range (e.g., 3 - 4).
Set Segment duration range (e.g., 3 - 4 seconds).
Toggle Randomize Segment Selection as desired.
Toggle Shuffle Output Segments as desired.
Check Enable GPU Acceleration if you have an NVIDIA GPU.
Click Start.
