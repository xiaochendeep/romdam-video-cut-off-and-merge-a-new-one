package main

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx       context.Context
	processor *Processor
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{}
	a.processor = NewProcessor(a)
	return a
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SelectFiles opens a file dialog to select multiple video files
func (a *App) SelectFiles() ([]string, error) {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Video Files",
		Filters: []runtime.FileFilter{
			{DisplayName: "Video Files (*.mp4;*.mov;*.mkv;*.ts;*.avi)", Pattern: "*.mp4;*.mov;*.mkv;*.ts;*.avi"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	return files, err
}

// SelectOutput opens a save dialog to select the output file path
func (a *App) SelectOutput() (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Select Output File",
		DefaultFilename: "output.mp4",
		Filters: []runtime.FileFilter{
			{DisplayName: "MP4 Files (*.mp4)", Pattern: "*.mp4"},
		},
	})
	return path, err
}

// StartProcessing triggers the video processing logic
func (a *App) StartProcessing(config Config) {
	a.processor.Start(config)
}

// AbortProcessing stops the current video processing task
func (a *App) AbortProcessing() {
	a.processor.Stop()
}

// EmitLog sends a log message to the frontend
func (a *App) EmitLog(message string) {
	runtime.EventsEmit(a.ctx, "log", message)
}

// EmitProgress sends a progress update to the frontend
func (a *App) EmitProgress(progress int) {
	runtime.EventsEmit(a.ctx, "progress", progress)
}

// EmitFinished sends a completion signal to the frontend
func (a *App) EmitFinished(success bool, message string) {
	runtime.EventsEmit(a.ctx, "finished", map[string]interface{}{
		"success": success,
		"message": message,
	})
}
