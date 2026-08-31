//go:build js

package verification

import (
	"fmt"
	"testing"
)

func TestEmitCheckpointForCrossTarget(t *testing.T) {
	records, err := ReferenceRun(referenceSeed, MaxTurns, "reference")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := CanonicalJSON(records)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("AFRICA2ICE-CHECKPOINT-BEGIN")
	fmt.Println(string(payload))
	fmt.Println("AFRICA2ICE-CHECKPOINT-END")
}
