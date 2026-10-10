# Ambient music listening prototype

Generate a two-minute instrumental piece with no recordings, dependencies, or
audio device:

```sh
mkdir -p dist
go run ./tools/generate_ambient -out dist/ambient-prototype.wav
```

The stereo 44.1 kHz, 16-bit WAV features a plucked melody in D natural minor.
Eight ten-note phrases last twelve seconds each, with a returning theme,
question-and-answer contours, small runs, and wider leaps. Two quieter bass and
fifth notes per phrase provide intermittent accompaniment. There are no sustained
drone oscillators. Karplus-Strong strings use a pitched excitation with a little
noise for a clear fundamental and wooden attack; notes have three-second tails.
Restrained stereo reverb adds space without dominating the tune. Two-second
opening and eight-second closing fades include the reverb. This is a finite
piece, not a seamless loop; the melody begins at 2.5 seconds.

Use `-seed 42` to vary the string excitations and stereo placement while keeping
the composition. Use `-duration 45s` for a shorter audition (1s to 30m accepted).
The generator streams to disk with bounded instrument and delay memory. Its
private random generator is independent of campaign state.

The tool is a listening prototype outside the game dependency graph. Audition
and tune it before moving synthesis into `pkg/audio`; continuous in-game music
will also need gesture-gated startup, preference settlement, focus handling,
and playback lifecycle support in the existing audio manager.
