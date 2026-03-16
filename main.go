package main

import (
	appbootstrap "codeswitch/internal/app/bootstrap"
	appdesktop "codeswitch/internal/app/desktop"
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/icon.png assets/icon-dark.png
var trayIcons embed.FS

func main() {
	container, err := appbootstrap.Initialize()
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}
	defer container.Shutdown()

	versionService := NewVersionService()
	app := application.New(application.Options{
		Name:        "Code Switch",
		Description: "Claude Code and Codex provier manager",
		Services: container.ApplicationServices(
			application.NewService(versionService),
		),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	container.LogsWindowService().SetApp(app)
	appdesktop.NewShell(app, container.DockService(), trayIcons).Configure()
	container.StartBackground()
	app.OnShutdown(container.Shutdown)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
