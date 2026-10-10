# Ambient music listening prototype

Generate a two-minute instrumental piece with no recordings, dependencies, or
audio device:

```sh
mkdir -p dist
go run ./tools/generate_ambient -out dist/ambient-prototype.wav
```

The stereo 44.1 kHz, 16-bit WAV contains softly detuned drones, moving bass roots,
Karplus-Strong plucked strings, and damped stereo reverb. Eight open chord
voicings change over 18-second sections, with an independent upper drone line
changing notes every nine seconds. Eight melodies span D natural minor across
several registers, mixing small steps, thirds through sevenths, octave leaps, and changing
contours over 14-second phrases, with deliberate gaps. Six-second opening and ten-second
closing fades include the reverb. This is a finite piece, not a seamless loop.

Use `-seed 42` to vary the string excitations and stereo placement while keeping
the composition. Use `-duration 45s` for a shorter audition (1s to 30m accepted).
The generator streams to disk with bounded instrument and delay memory. Its
private random generator is independent of campaign state.

The tool is a listening prototype outside the game dependency graph. Audition
and tune it before moving synthesis into `pkg/audio`; continuous in-game music
will also need gesture-gated startup, preference settlement, focus handling,
and playback lifecycle support in the existing audio manager.
