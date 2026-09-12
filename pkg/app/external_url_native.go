//go:build !js

package app

import (
	"fmt"
	"os/exec"
	"runtime"
)

// externalURLCommand delegates to the operating system's default URL handler.
// Arguments go directly to the executable, never through a shell.
func externalURLCommand(platform, url string) (*exec.Cmd, error) {
	switch platform {
	case "darwin":
		return exec.Command("open", url), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url), nil
	case "linux", "freebsd", "openbsd", "netbsd":
		return exec.Command("xdg-open", url), nil
	default:
		return nil, fmt.Errorf("unsupported browser platform: %s", platform)
	}
}

func openExternalURL(url string) error {
	command, err := externalURLCommand(runtime.GOOS, url)
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}
	// Reap the launcher without stalling the game while the browser starts.
	go func() { _ = command.Wait() }()
	return nil
}
