package main

import (
	"embed"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// The built frontend is embedded in the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	settings := loadSettings()
	service := &LogService{settings: settings, version: version}
	app := application.New(application.Options{
		Name:        "LogCrawler",
		Description: "Reads the ServiceDesk Connect diagnostics log as a tree",
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	start := application.WindowCentered
	if settings.X != 0 || settings.Y != 0 {
		start = application.WindowXY
	}
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		InitialPosition: start,
		Title:           "LogCrawler " + version,
		Width:           settings.Width,
		Height:          settings.Height,
		X:               settings.X,
		Y:               settings.Y,
		MinWidth:        640,
		MinHeight:       400,
		EnableFileDrop:  true,
		URL:             "/",
	})

	// A dropped file is opened by the frontend, which knows how to show the result or the error.
	win.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		if files := e.Context().DroppedFiles(); len(files) > 0 {
			app.Event.Emit("file-dropped", files[0])
		}
	})

	// Remember the window size and place for next time.
	remember := func(*application.WindowEvent) {
		settings.mu.Lock()
		settings.Width, settings.Height = win.Size()
		settings.X, settings.Y = win.Position()
		settings.mu.Unlock()
	}
	win.OnWindowEvent(events.Common.WindowDidResize, remember)
	win.OnWindowEvent(events.Common.WindowDidMove, remember)
	win.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		remember(e)
		settings.save()
	})

	// Tell the window when the open file changes on disk.
	go func() {
		for range time.Tick(2 * time.Second) {
			service.pollChanged()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
