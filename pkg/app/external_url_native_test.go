//go:build !js

package app

import (
	"reflect"
	"testing"
)

func TestExternalURLUsesPlatformHandler(t *testing.T) {
	const url = "https://doi.org/10.1038/nature09710"
	for platform, want := range map[string][]string{
		"darwin":  {"open", url},
		"windows": {"rundll32", "url.dll,FileProtocolHandler", url},
		"linux":   {"xdg-open", url},
	} {
		command, err := externalURLCommand(platform, url)
		if err != nil || !reflect.DeepEqual(command.Args, want) {
			t.Fatalf("%s: command %v, error %v", platform, command, err)
		}
	}
	if _, err := externalURLCommand("unsupported", url); err == nil {
		t.Fatal("unsupported platform accepted")
	}
}
