//go:build !js

package main

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/app"
)

func TestInitialDesktopWindowSize(t *testing.T) {
	tests := []struct {
		name                  string
		monitorWidth          int
		monitorHeight         int
		wantWidth, wantHeight int
	}{
		{name: "full HD", monitorWidth: 1920, monitorHeight: 1080, wantWidth: 1728, wantHeight: 972},
		{name: "4K", monitorWidth: 3840, monitorHeight: 2160, wantWidth: 3456, wantHeight: 1944},
		{name: "ultrawide is height limited", monitorWidth: 3440, monitorHeight: 1440, wantWidth: 2304, wantHeight: 1296},
		{name: "invalid monitor falls back", monitorWidth: 0, monitorHeight: 0, wantWidth: app.LogicalWidth, wantHeight: app.LogicalHeight},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			width, height := initialDesktopWindowSize(test.monitorWidth, test.monitorHeight)
			if width != test.wantWidth || height != test.wantHeight {
				t.Fatalf("initialDesktopWindowSize(%d, %d) = %dx%d, want %dx%d", test.monitorWidth, test.monitorHeight, width, height, test.wantWidth, test.wantHeight)
			}
		})
	}
}
