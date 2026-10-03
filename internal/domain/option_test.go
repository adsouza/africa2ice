package domain

import (
	"reflect"
	"testing"
)

func TestOptionZeroIsAbsentAndSomeHoldsItsValue(t *testing.T) {
	var none Option[TileID]
	if _, ok := none.Get(); ok || none.Present() {
		t.Fatal("zero Option is present")
	}
	some := Some(TileID(7))
	if value, ok := some.Get(); !ok || value != 7 || !some.Present() {
		t.Fatalf("Some(7).Get() = %v, %v", value, ok)
	}
}

// hasReference reports whether values of the type share memory when copied.
func hasReference(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer, reflect.String:
		return typ.Kind() != reflect.String // strings are immutable, so sharing is safe
	case reflect.Array:
		return hasReference(typ.Elem())
	case reflect.Struct:
		for index := range typ.NumField() {
			if hasReference(typ.Field(index).Type) {
				return true
			}
		}
	}
	return false
}

// World.Bands() and ExportState copy bands by value; that isolates the live
// aggregate only while Band holds nothing a copy would share. A pointer-based
// optional order would quietly break it.
func TestBandHoldsNoReferenceFields(t *testing.T) {
	if !hasReference(reflect.TypeOf(struct{ order *MigrationOrder }{})) {
		t.Fatal("the reference check cannot detect a pointer, so it proves nothing")
	}
	if hasReference(reflect.TypeOf(Band{})) {
		t.Fatal("Band holds a reference field; copies returned by World.Bands() would alias the aggregate")
	}
}

// An accepted interbreeding intent resolves at the end of the turn and leaves
// no target behind, not a stale ID beside a false flag.
func TestResolvedInterbreedingLeavesNoTarget(t *testing.T) {
	world, err := NewWorld(1)
	if err != nil {
		t.Fatal(err)
	}
	sapiens, archaic := -1, -1
	for index, band := range world.bands {
		switch {
		case band.Species == HomoSapiens && sapiens < 0:
			sapiens = index
		case band.Species == ArchaicHominin && archaic < 0:
			archaic = index
		}
	}
	world.bands[archaic].TileID = world.bands[sapiens].TileID
	actor, target := world.bands[sapiens].ID, world.bands[archaic].ID
	if err := world.Interbreed(actor, target, true); err != nil {
		t.Fatal(err)
	}
	if got, ok := world.bands[sapiens].InterbreedTarget.Get(); !ok || got != target {
		t.Fatalf("accepted target = %v, %v; want %v", got, ok, target)
	}
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	for _, band := range world.bands {
		if band.ID == actor && band.InterbreedTarget != (Option[BandID]{}) {
			t.Fatalf("resolved band kept interbreed target %+v", band.InterbreedTarget)
		}
	}
}
