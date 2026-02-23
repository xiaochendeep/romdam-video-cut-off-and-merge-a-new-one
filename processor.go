package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	Files           []string `json:"files"`
	CountMin        int      `json:"count_min"`
	CountMax        int      `json:"count_max"`
	SegmentMin      int      `json:"segment_min"`
	SegmentMax      int      `json:"segment_max"`
	StartOffsetMin  int      `json:"start_offset_min"`
	RandomTime      bool     `json:"random_time"`
	ShuffleSegments bool     `json:"shuffle_segments"`
	GPU             bool     `json:"gpu"`
	OutputPath      string   `json:"output_path"`
}

type Processor struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	app    *App
}

func NewProcessor(app *App) *Processor {
	return &Processor{app: app}
}

func (p *Processor) Start(config Config) {
	p.mu.Lock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.mu.Unlock()

	go p.run(config)
}

func (p *Processor) Stop() {
	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
	}
	p.mu.Unlock()
}

func (p *Processor) log(msg string) {
	p.app.EmitLog(msg)
}

func (p *Processor) progress(pct int) {
	p.app.EmitProgress(pct)
}

func (p *Processor) run(config Config) {
	defer func() {
		if r := recover(); r != nil {
			p.log(fmt.Sprintf("Panic: %v", r))
			p.app.EmitFinished(false, fmt.Sprintf("Critical error: %v", r))
		}
	}()

	p.log(fmt.Sprintf("Starting processing %d files...", len(config.Files)))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	var segRequests []SegmentRequest
	minStartOffset := float64(config.StartOffsetMin) * 60.0

	for _, file := range config.Files {
		duration, err := GetVideoDuration(file)
		if err != nil {
			p.log(fmt.Sprintf("Skipping %s: %v", file, err))
			continue
		}

		if duration <= minStartOffset+1.0 {
			p.log(fmt.Sprintf("Skipping %s: too short", file))
			continue
		}

		n := r.Intn(config.CountMax-config.CountMin+1) + config.CountMin
		var chosen []struct {
			start float64
			dur   float64
		}

		if !config.RandomTime {
			currentStart := minStartOffset
			for i := 0; i < n; i++ {
				segLen := float64(config.SegmentMin+config.SegmentMax) / 2.0
				if currentStart+segLen > duration {
					break
				}
				chosen = append(chosen, struct {
					start float64
					dur   float64
				}{currentStart, segLen})
				currentStart += segLen
			}
		} else {
			// Basic random selection without overlap
			available := duration - minStartOffset
			if available < float64(config.SegmentMax) {
				continue
			}
			
			for i := 0; i < n; i++ {
				segLen := r.Float64()*float64(config.SegmentMax-config.SegmentMin) + float64(config.SegmentMin)
				if duration-minStartOffset < segLen {
					break
				}
				start := r.Float64()*(duration-minStartOffset-segLen) + minStartOffset
				chosen = append(chosen, struct {
					start float64
					dur   float64
				}{start, segLen})
			}
		}

		stem := filepath.Base(file)
		for i, c := range chosen {
			outName := fmt.Sprintf("%s_seg_%d_%d.mp4", stem, i, time.Now().UnixNano())
			outPath := filepath.Join(os.TempDir(), "video_processor_segments", outName)
			segRequests = append(segRequests, SegmentRequest{
				SrcPath:  file,
				Start:    c.start,
				Duration: c.dur,
				OutPath:  outPath,
			})
		}
	}

	if len(segRequests) == 0 {
		p.app.EmitFinished(false, "No segments to process")
		return
	}

	if config.ShuffleSegments {
		r.Shuffle(len(segRequests), func(i, j int) {
			segRequests[i], segRequests[j] = segRequests[j], segRequests[i]
		})
	}

	// Parallel extraction
	var wg sync.WaitGroup
	limit := make(chan struct{}, 4)
	doneCount := 0
	var doneMu sync.Mutex

	for _, req := range segRequests {
		select {
		case <-p.ctx.Done():
			p.app.EmitFinished(false, "Cancelled")
			return
		default:
		}

		wg.Add(1)
		go func(r SegmentRequest) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()

			_, err := RunFFmpegExtract(r, config.GPU)
			doneMu.Lock()
			doneCount++
			p.progress(int(float64(doneCount) / float64(len(segRequests)) * 80))
			if err != nil {
				p.log(fmt.Sprintf("Failed to extract %s: %v", r.SrcPath, err))
			} else {
				p.log(fmt.Sprintf("Extracted segment: %s", filepath.Base(r.OutPath)))
			}
			doneMu.Unlock()
		}(req)
	}

	wg.Wait()

	// Concat
	var existing []string
	for _, req := range segRequests {
		if _, err := os.Stat(req.OutPath); err == nil {
			existing = append(existing, req.OutPath)
		}
	}

	if len(existing) == 0 {
		p.app.EmitFinished(false, "Extraction failed for all segments")
		return
	}

	p.log("Concatenating segments...")
	_, err := ConcatSegments(existing, config.OutputPath, config.GPU)
	if err != nil {
		p.app.EmitFinished(false, fmt.Sprintf("Concat failed: %v", err))
		return
	}

	p.progress(100)
	p.app.EmitFinished(true, config.OutputPath)
	
	// Cleanup temp segments
	for _, s := range existing {
		os.Remove(s)
	}
}
