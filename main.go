package main

import (
	"context"
	"embed"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"agentdesk/internal/skills"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:         "Agent Desk",
		Width:         1200,
		Height:        800,
		MinWidth:      900,
		MinHeight:     600,
		DisableResize: false,
		AssetServer:   &assetserver.Options{Assets: assets},
		OnStartup:     app.startup,
		OnShutdown:    app.shutdown,
		OnBeforeClose: func(_ context.Context) (prevent bool) {
			// On Linux and Windows, if "Close to background" is enabled, hide the
			// window instead of quitting when the user closes it.
			if goruntime.GOOS == "darwin" {
				return false // macOS apps conventionally quit on window close
			}
			s := skills.LoadSettings()
			if s.RunInBackground {
				app.Hide()
				return true // prevent the default close
			}
			return false // allow normal close
		},
		Bind: []interface{}{app},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About:    &mac.AboutInfo{Title: "Agent Desk", Message: "AI agent skill file manager"},
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
