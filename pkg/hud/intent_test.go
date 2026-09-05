package hud

import "testing"

// TestIntentKindStringIsDistinctAndNonEmpty covers Wave I item I0: the UI
// observability log speaks IntentKind's String() as its vocabulary, so every
// kind the chrome can actually emit needs a stable, non-empty, distinct
// name — a collision or a blank name would make two different player
// actions indistinguishable in the log.
func TestIntentKindStringIsDistinctAndNonEmpty(t *testing.T) {
	seen := make(map[string]IntentKind, IntentKindCount)
	for kind := IntentKind(0); kind < IntentKindCount; kind++ {
		name := kind.String()
		if name == "" {
			t.Fatalf("IntentKind(%d).String() is empty", kind)
		}
		if other, ok := seen[name]; ok {
			t.Fatalf("IntentKind(%d) and IntentKind(%d) both stringify to %q", kind, other, name)
		}
		seen[name] = kind
	}
}
