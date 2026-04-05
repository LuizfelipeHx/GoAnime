package main

import (
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// SendDesktopNotification emits a notification event to the frontend.
// The frontend can display it as a toast or system notification.
func (a *App) SendDesktopNotification(title, body string) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "notification", map[string]interface{}{
		"title": title,
		"body":  body,
	})
}
