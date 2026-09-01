// Command generate_compatibility_fixture writes the frozen schema-v1 save used
// by adapter compatibility tests. Regenerate it only when intentionally
// changing the oldest supported durable representation.
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"time"

	"github.com/adsouza/africa2ice/internal/application"
)

const (
	fixtureSeed  = uint64(0x9e3779b97f4a7c15)
	fixtureTurns = 12
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/generate_compatibility_fixture OUTPUT.json.gz")
		os.Exit(2)
	}
	service, err := application.NewGameService(fixtureSeed)
	if err != nil {
		fatal(err)
	}
	for range fixtureTurns {
		if _, err := service.EndTurn(); err != nil {
			fatal(err)
		}
	}
	state, err := service.ExportSaveState()
	if err != nil {
		fatal(err)
	}
	payload, err := application.EncodeSaveState(state)
	if err != nil {
		fatal(err)
	}
	var compressed bytes.Buffer
	writer, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		fatal(err)
	}
	writer.ModTime = time.Unix(0, 0).UTC()
	writer.OS = 255
	if _, err := writer.Write(payload); err != nil {
		fatal(err)
	}
	if err := writer.Close(); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(os.Args[1], compressed.Bytes(), 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
