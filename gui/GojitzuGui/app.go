package main

import (
	"context"
	"fmt"
	"os"

	"github.com/defektive/gojitzu/cmd"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	home, _ := os.UserHomeDir()
	os.Chdir(home)

	runtime.EventsEmit(a.ctx, "log", "✅ Wails backend connected\n")
	runtime.EventsEmit(a.ctx, "log", "Running from: "+home+"\n")
}

func (a *App) RunGojitzu(args []string) string {
	output, err := cmd.RunForGUI(args)
	if err != nil {
		return output + "\nERROR: " + err.Error()
	}
	return output
}

func (a *App) RunGojitzuAsync(args []string) {

	runtime.EventsEmit(a.ctx, "log", "Executing: gojitzu "+fmt.Sprint(args)+"\n\n")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				runtime.EventsEmit(a.ctx, "log", "🔥 PANIC: "+fmt.Sprint(r)+"\n")
			}
		}()

		fullArgs := append([]string{"tpl"}, args...)

		output, err := cmd.RunForGUI(fullArgs)

		if err != nil {
			runtime.EventsEmit(a.ctx, "log", "❌ ERROR: "+err.Error()+"\n")
		}

		if output != "" {
			runtime.EventsEmit(a.ctx, "log", output)
		}
	}()
}

func (a *App) GetProjects() map[string]interface{} {
	projects, debug, err := cmd.GetProjects()

	result := map[string]interface{}{
		"projects": projects,
		"debug":    debug,
		"error":    "",
	}

	if err != nil {
		result["error"] = err.Error()
	}

	return result
}

func (a *App) GetEpics(projectKey string) map[string]interface{} {
	epics, debug, err := cmd.GetEpics(projectKey)

	result := map[string]interface{}{
		"epics": epics,
		"debug": debug,
		"error": "",
	}

	if err != nil {
		result["error"] = err.Error()
	}

	return result
}

func (a *App) GetTemplates() map[string]interface{} {
	templates, err := cmd.GetTemplates()

	result := map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
		"error":     "",
	}

	if err != nil {
		result["error"] = err.Error()
	}

	return result
}

func (a *App) GetDefaultProject() string {
	return cmd.GetDefaultProject()
}
