package desktop

import (
	"fmt"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type LogsWindowService struct {
	App *application.App
}

func NewLogsWindowService() *LogsWindowService {
	return &LogsWindowService{}
}

func (s *LogsWindowService) SetApp(app *application.App) {
	s.App = app
}

func (s *LogsWindowService) OpenSecondWindow() {
	if s.App == nil {
		log.Println("[ERROR] app not initialized")
		return
	}
	name := fmt.Sprintf("logs-%d", time.Now().UnixNano())
	win := s.App.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Logs",
		Name:      name,
		Width:     1024,
		Height:    800,
		MinWidth:  600,
		MinHeight: 300,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			TitleBar:                application.MacTitleBarHidden,
			Backdrop:                application.MacBackdropTransparent,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/#/logs",
	})
	win.Center()
}
