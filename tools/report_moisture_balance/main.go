package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/adsouza/africa2ice/internal/application"
)

func main() {
	report, err := application.BuildMoistureBalanceReport()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := application.ValidateMoistureBalanceReport(report); err != nil {
		fmt.Fprintln(os.Stderr, "moisture balance gate:", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
