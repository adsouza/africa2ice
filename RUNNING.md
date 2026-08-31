# Running the portable build

Extract the archive and run `africa2ice` (`africa2ice.exe` on Windows). The v1 archives are
unsigned. macOS, Windows, or Linux may therefore display an unrecognized-developer warning; inspect
the archive against the published `SHA256SUMS`, then use the operating system's normal one-time
override if you trust the download.

The executable needs a graphical desktop for ordinary play. For a display-free integrity smoke,
run `africa2ice -headless -turns 0`.

See the [repository README](https://github.com/adsouza/africa2ice#readme) for controls, saves, logs,
and web-build instructions.
