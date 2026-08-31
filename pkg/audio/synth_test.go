package audio

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestSynthesizedEffectsAreBoundedStereoAndDistinct(t *testing.T) {
	wantBytes := int(sampleRate*soundDurationSec) * channelCount * bytesPerSample
	var zeroCrossings [soundCount]int
	for sound := Sound(0); sound < soundCount; sound++ {
		pcm := synthPCM(sound)
		if len(pcm) != wantBytes {
			t.Fatalf("sound %d bytes = %d, want %d", sound, len(pcm), wantBytes)
		}
		var previous float32
		for frame := 0; frame < len(pcm)/(channelCount*bytesPerSample); frame++ {
			offset := frame * channelCount * bytesPerSample
			left := math.Float32frombits(binary.LittleEndian.Uint32(pcm[offset : offset+4]))
			right := math.Float32frombits(binary.LittleEndian.Uint32(pcm[offset+4 : offset+8]))
			if left != right || math.IsNaN(float64(left)) || math.IsInf(float64(left), 0) || math.Abs(float64(left)) > 1 {
				t.Fatalf("sound %d frame %d = (%v,%v)", sound, frame, left, right)
			}
			if previous <= 0 && left > 0 {
				zeroCrossings[sound]++
			}
			previous = left
		}
		if first := math.Float32frombits(binary.LittleEndian.Uint32(pcm[:4])); first != 0 {
			t.Fatalf("sound %d starts at %v, want silent envelope edge", sound, first)
		}
	}
	if zeroCrossings[SFXChoiceClick] >= zeroCrossings[SFXSaveComplete] || zeroCrossings[SFXSaveComplete] >= zeroCrossings[SFXEventTrigger] {
		t.Fatalf("pitch ordering by zero crossings = %v", zeroCrossings)
	}
}

func TestNoopAndLazyMasterSettingsAreSafeBeforeConstruction(t *testing.T) {
	NoopManager{}.SetMaster(0.8, true)
	NoopManager{}.Play(SFXChoiceClick)
	lazy := NewLazyManager()
	lazy.SetMaster(2, true)
	if lazy.volume != 1 || !lazy.muted || !lazy.settled || lazy.manager != nil {
		t.Fatalf("lazy settings = volume %v muted %t manager %T", lazy.volume, lazy.muted, lazy.manager)
	}
	lazy.SetMaster(-1, false)
	if lazy.volume != 0 || lazy.muted {
		t.Fatalf("clamped lazy settings = volume %v muted %t", lazy.volume, lazy.muted)
	}
}
