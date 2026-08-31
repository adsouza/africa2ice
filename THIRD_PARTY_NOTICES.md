# Third-party notices

Africa 2 Ice includes or depends on the following third-party software. Versions are locked by
`go.mod`, `go.sum`, and `tools/web-e2e/package-lock.json`.

## Software shipped in the desktop and WebAssembly builds

| Component | License |
| --- | --- |
| Ebitengine and `hideconsole`/`oto`/`purego` support modules | Apache License 2.0 |
| `github.com/go-text/typesetting` | BSD-3-Clause or Unlicense |
| `github.com/rivo/uniseg` | MIT License |
| `golang.org/x/image`, `x/sync`, `x/sys`, and `x/text` | BSD-3-Clause |

The complete copyright and license texts for these modules are distributed in their upstream source
repositories and in the Go module cache used to build this program. The corresponding upstream
project URLs and exact versions are recorded by `go list -m all`.

## Bundled Go Regular font

The UI embeds Go Regular from the Go font family, created by Bigelow & Holmes for the Go project.

Copyright (c) 2016 Bigelow & Holmes Inc. All rights reserved.

Redistribution and use in source and binary forms, with or without modification, are permitted
provided that source redistributions retain the copyright notice, conditions, and disclaimer;
binary redistributions reproduce them in accompanying materials; and neither Google Inc. nor
contributors' names are used to endorse derived products without permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS", WITHOUT EXPRESS OR
IMPLIED WARRANTIES, INCLUDING MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE. IN NO EVENT
SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
EXEMPLARY, OR CONSEQUENTIAL DAMAGES ARISING IN ANY WAY FROM ITS USE.

## Development and release tooling

Playwright is used only by browser verification and is not shipped in the game. It is licensed under
the Apache License 2.0. Binaryen is release-build tooling and is licensed under the Apache License
2.0. Their exact versions are pinned in the repository.
