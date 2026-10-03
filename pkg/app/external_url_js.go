//go:build js && wasm

package app

import "syscall/js"

func openExternalURL(url string, _ func()) error {
	js.Global().Call("open", url, "_blank", "noopener,noreferrer")
	return nil
}
