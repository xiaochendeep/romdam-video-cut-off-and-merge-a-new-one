package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// SegmentRequest defines a video segment extraction request
type SegmentRequest struct {
	SrcPath  string  `json:"src_path"`
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
	OutPath  string  `json:"out_path"`
}

// GetVideoDuration uses ffprobe to get the duration of a video file
func GetVideoDuration(path string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %v, output: %s", err, string(out))
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}
	return duration, nil
}

// RunFFmpegExtract extracts a segment from a video
func RunFFmpegExtract(req SegmentRequest, useGPU bool) (string, error) {
	outDir := filepath.Dir(req.OutPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	var vcodec string
	var preset string
	if useGPU {
		vcodec = "h264_nvenc"
		preset = "p7"
	} else {
		vcodec = "libx264"
		preset = "fast"
	}

	// filters for standardization: 1080p, 30fps, aspect ratio preservation
	filters := "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2"

	args := []string{
		"-y", "-err_detect", "ignore_err", "-ignore_unknown",
		"-ss", fmt.Sprintf("%.3f", req.Start),
		"-i", req.SrcPath,
		"-t", fmt.Sprintf("%.3f", req.Duration),
		"-vf", filters,
		"-r", "30",
		"-c:v", vcodec,
		"-preset", preset,
		"-c:a", "aac",
		"-ar", "44100",
		"-ac", "2",
		"-pix_fmt", "yuv420p",
		req.OutPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stderr.String(), fmt.Errorf("ffmpeg extract failed: %v", err)
	}
	return stderr.String(), nil
}

// ConcatSegments merges multiple video segments into one
func ConcatSegments(segmentFiles []string, outPath string, useGPU bool) (string, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", "concat_")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	listPath := filepath.Join(tmpDir, "list.txt")
	f, err := os.Create(listPath)
	if err != nil {
		return "", err
	}
	for _, s := range segmentFiles {
		absPath, _ := filepath.Abs(s)
		fmt.Fprintf(f, "file '%s'\n", strings.ReplaceAll(absPath, "\\", "/"))
	}
	f.Close()

	var vcodec string
	var preset string
	if useGPU {
		vcodec = "h264_nvenc"
		preset = "p7"
	} else {
		vcodec = "libx264"
		preset = "medium"
	}

	args := []string{
		"-y", "-f", "concat", "-safe", "0", "-i", listPath,
		"-c:v", vcodec, "-preset", preset,
		"-c:a", "aac",
		outPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stderr.String(), fmt.Errorf("ffmpeg concat failed: %v", err)
	}
	return stderr.String(), nil
}
