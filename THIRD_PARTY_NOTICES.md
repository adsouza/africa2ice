# Third-party notices

Africa 2 Ice includes or uses the software below. This inventory was reviewed against
`go list -m all`, `tools/web-e2e/package-lock.json`, the pinned Binaryen release, and the bundled
font source. `go run ./tools/check_audits` rejects missing, stale, or version-mismatched inventory
rows. “Release import graph” means at least one current desktop or WebAssembly release target
imports the module; “complete Go build list only” means the module is selected by Go's module graph
but no current release target imports one of its packages.

License classifications below come from the top-level license file shipped with each exact module
or package version. The linked upstream records provide the corresponding copyright and license
text. This audit is an attribution and release-review record, not legal advice.

## Go module build list

| Ecosystem | Component | Version | License | Current scope | Upstream record |
| --- | --- | --- | --- | --- | --- |
| Go | `github.com/ebitengine/debugui` | `v0.2.0` | Apache-2.0 | Complete Go build list only | [source](https://pkg.go.dev/github.com/ebitengine/debugui@v0.2.0?tab=licenses) |
| Go | `github.com/ebitengine/gomobile` | `v0.0.0-20250923094054-ea854a63cce1` | BSD-3-Clause | Complete Go build list only | [source](https://github.com/ebitengine/gomobile/blob/ea854a63cce1/LICENSE) |
| Go | `github.com/ebitengine/hideconsole` | `v1.0.0` | Apache-2.0 | Release import graph | [source](https://pkg.go.dev/github.com/ebitengine/hideconsole@v1.0.0?tab=licenses) |
| Go | `github.com/ebitengine/oto/v3` | `v3.4.1` | Apache-2.0 | Release import graph | [source](https://pkg.go.dev/github.com/ebitengine/oto/v3@v3.4.1?tab=licenses) |
| Go | `github.com/ebitengine/purego` | `v0.9.1` | Apache-2.0 | Release import graph | [source](https://pkg.go.dev/github.com/ebitengine/purego@v0.9.1?tab=licenses) |
| Go | `github.com/gen2brain/mpeg` | `v0.5.0` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/gen2brain/mpeg@v0.5.0?tab=licenses) |
| Go | `github.com/go-text/typesetting` | `v0.3.0` | Unlicense OR BSD-3-Clause | Release import graph | [source](https://pkg.go.dev/github.com/go-text/typesetting@v0.3.0?tab=licenses) |
| Go | `github.com/go-text/typesetting-utils` | `v0.0.0-20241103174707-87a29e9e6066` | Unlicense OR BSD-3-Clause | Complete Go build list only | [source](https://github.com/go-text/typesetting-utils/blob/87a29e9e6066/LICENSE) |
| Go | `github.com/hajimehoshi/bitmapfont/v4` | `v4.1.0` | Apache-2.0 | Complete Go build list only | [source](https://pkg.go.dev/github.com/hajimehoshi/bitmapfont/v4@v4.1.0?tab=licenses) |
| Go | `github.com/hajimehoshi/ebiten/v2` | `v2.9.10` | Apache-2.0 | Release import graph | [source](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2@v2.9.10?tab=licenses) |
| Go | `github.com/hajimehoshi/go-mp3` | `v0.3.4` | Apache-2.0 | Complete Go build list only | [source](https://pkg.go.dev/github.com/hajimehoshi/go-mp3@v0.3.4?tab=licenses) |
| Go | `github.com/jakecoffman/cp/v2` | `v2.3.0` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/jakecoffman/cp/v2@v2.3.0?tab=licenses) |
| Go | `github.com/jezek/xgb` | `v1.2.0` | BSD-3-Clause | Release import graph | [source](https://pkg.go.dev/github.com/jezek/xgb@v1.2.0?tab=licenses) |
| Go | `github.com/jfreymuth/oggvorbis` | `v1.0.5` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/jfreymuth/oggvorbis@v1.0.5?tab=licenses) |
| Go | `github.com/jfreymuth/vorbis` | `v1.0.2` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/jfreymuth/vorbis@v1.0.2?tab=licenses) |
| Go | `github.com/kisielk/errcheck` | `v1.9.0` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/kisielk/errcheck@v1.9.0?tab=licenses) |
| Go | `github.com/pierrec/lz4/v4` | `v4.1.22` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/github.com/pierrec/lz4/v4@v4.1.22?tab=licenses) |
| Go | `github.com/rivo/uniseg` | `v0.4.7` | MIT | Release import graph | [source](https://pkg.go.dev/github.com/rivo/uniseg@v0.4.7?tab=licenses) |
| Go | `golang.org/x/image` | `v0.43.0` | BSD-3-Clause | Release import graph | [source](https://pkg.go.dev/golang.org/x/image@v0.43.0?tab=licenses) |
| Go | `golang.org/x/mod` | `v0.36.0` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/golang.org/x/mod@v0.36.0?tab=licenses) |
| Go | `golang.org/x/sync` | `v0.21.0` | BSD-3-Clause | Release import graph | [source](https://pkg.go.dev/golang.org/x/sync@v0.21.0?tab=licenses) |
| Go | `golang.org/x/sys` | `v0.44.0` | BSD-3-Clause | Release import graph | [source](https://pkg.go.dev/golang.org/x/sys@v0.44.0?tab=licenses) |
| Go | `golang.org/x/text` | `v0.38.0` | BSD-3-Clause | Release import graph | [source](https://pkg.go.dev/golang.org/x/text@v0.38.0?tab=licenses) |
| Go | `golang.org/x/tools` | `v0.45.0` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/golang.org/x/tools@v0.45.0?tab=licenses) |
| Go | `golang.org/x/tools/go/expect` | `v0.1.1-deprecated` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/golang.org/x/tools/go/expect@v0.1.1-deprecated?tab=licenses) |
| Go | `golang.org/x/tools/go/packages/packagestest` | `v0.1.1-deprecated` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/golang.org/x/tools/go/packages/packagestest@v0.1.1-deprecated?tab=licenses) |

## License texts for release-imported modules

### Apache License 2.0

This text applies to `github.com/ebitengine/hideconsole`, `github.com/ebitengine/oto/v3`,
`github.com/ebitengine/purego`, and `github.com/hajimehoshi/ebiten/v2`. None of those exact module
versions contains an additional top-level `NOTICE` file.

```text
                                 Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/

   TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

   1. Definitions.

      "License" shall mean the terms and conditions for use, reproduction,
      and distribution as defined by Sections 1 through 9 of this document.

      "Licensor" shall mean the copyright owner or entity authorized by
      the copyright owner that is granting the License.

      "Legal Entity" shall mean the union of the acting entity and all
      other entities that control, are controlled by, or are under common
      control with that entity. For the purposes of this definition,
      "control" means (i) the power, direct or indirect, to cause the
      direction or management of such entity, whether by contract or
      otherwise, or (ii) ownership of fifty percent (50%) or more of the
      outstanding shares, or (iii) beneficial ownership of such entity.

      "You" (or "Your") shall mean an individual or Legal Entity
      exercising permissions granted by this License.

      "Source" form shall mean the preferred form for making modifications,
      including but not limited to software source code, documentation
      source, and configuration files.

      "Object" form shall mean any form resulting from mechanical
      transformation or translation of a Source form, including but
      not limited to compiled object code, generated documentation,
      and conversions to other media types.

      "Work" shall mean the work of authorship, whether in Source or
      Object form, made available under the License, as indicated by a
      copyright notice that is included in or attached to the work
      (an example is provided in the Appendix below).

      "Derivative Works" shall mean any work, whether in Source or Object
      form, that is based on (or derived from) the Work and for which the
      editorial revisions, annotations, elaborations, or other modifications
      represent, as a whole, an original work of authorship. For the purposes
      of this License, Derivative Works shall not include works that remain
      separable from, or merely link (or bind by name) to the interfaces of,
      the Work and Derivative Works thereof.

      "Contribution" shall mean any work of authorship, including
      the original version of the Work and any modifications or additions
      to that Work or Derivative Works thereof, that is intentionally
      submitted to Licensor for inclusion in the Work by the copyright owner
      or by an individual or Legal Entity authorized to submit on behalf of
      the copyright owner. For the purposes of this definition, "submitted"
      means any form of electronic, verbal, or written communication sent
      to the Licensor or its representatives, including but not limited to
      communication on electronic mailing lists, source code control systems,
      and issue tracking systems that are managed by, or on behalf of, the
      Licensor for the purpose of discussing and improving the Work, but
      excluding communication that is conspicuously marked or otherwise
      designated in writing by the copyright owner as "Not a Contribution."

      "Contributor" shall mean Licensor and any individual or Legal Entity
      on behalf of whom a Contribution has been received by Licensor and
      subsequently incorporated within the Work.

   2. Grant of Copyright License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      copyright license to reproduce, prepare Derivative Works of,
      publicly display, publicly perform, sublicense, and distribute the
      Work and such Derivative Works in Source or Object form.

   3. Grant of Patent License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      (except as stated in this section) patent license to make, have made,
      use, offer to sell, sell, import, and otherwise transfer the Work,
      where such license applies only to those patent claims licensable
      by such Contributor that are necessarily infringed by their
      Contribution(s) alone or by combination of their Contribution(s)
      with the Work to which such Contribution(s) was submitted. If You
      institute patent litigation against any entity (including a
      cross-claim or counterclaim in a lawsuit) alleging that the Work
      or a Contribution incorporated within the Work constitutes direct
      or contributory patent infringement, then any patent licenses
      granted to You under this License for that Work shall terminate
      as of the date such litigation is filed.

   4. Redistribution. You may reproduce and distribute copies of the
      Work or Derivative Works thereof in any medium, with or without
      modifications, and in Source or Object form, provided that You
      meet the following conditions:

      (a) You must give any other recipients of the Work or
          Derivative Works a copy of this License; and

      (b) You must cause any modified files to carry prominent notices
          stating that You changed the files; and

      (c) You must retain, in the Source form of any Derivative Works
          that You distribute, all copyright, patent, trademark, and
          attribution notices from the Source form of the Work,
          excluding those notices that do not pertain to any part of
          the Derivative Works; and

      (d) If the Work includes a "NOTICE" text file as part of its
          distribution, then any Derivative Works that You distribute must
          include a readable copy of the attribution notices contained
          within such NOTICE file, excluding those notices that do not
          pertain to any part of the Derivative Works, in at least one
          of the following places: within a NOTICE text file distributed
          as part of the Derivative Works; within the Source form or
          documentation, if provided along with the Derivative Works; or,
          within a display generated by the Derivative Works, if and
          wherever such third-party notices normally appear. The contents
          of the NOTICE file are for informational purposes only and
          do not modify the License. You may add Your own attribution
          notices within Derivative Works that You distribute, alongside
          or as an addendum to the NOTICE text from the Work, provided
          that such additional attribution notices cannot be construed
          as modifying the License.

      You may add Your own copyright statement to Your modifications and
      may provide additional or different license terms and conditions
      for use, reproduction, or distribution of Your modifications, or
      for any such Derivative Works as a whole, provided Your use,
      reproduction, and distribution of the Work otherwise complies with
      the conditions stated in this License.

   5. Submission of Contributions. Unless You explicitly state otherwise,
      any Contribution intentionally submitted for inclusion in the Work
      by You to the Licensor shall be under the terms and conditions of
      this License, without any additional terms or conditions.
      Notwithstanding the above, nothing herein shall supersede or modify
      the terms of any separate license agreement you may have executed
      with Licensor regarding such Contributions.

   6. Trademarks. This License does not grant permission to use the trade
      names, trademarks, service marks, or product names of the Licensor,
      except as required for reasonable and customary use in describing the
      origin of the Work and reproducing the content of the NOTICE file.

   7. Disclaimer of Warranty. Unless required by applicable law or
      agreed to in writing, Licensor provides the Work (and each
      Contributor provides its Contributions) on an "AS IS" BASIS,
      WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
      implied, including, without limitation, any warranties or conditions
      of TITLE, NON-INFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A
      PARTICULAR PURPOSE. You are solely responsible for determining the
      appropriateness of using or redistributing the Work and assume any
      risks associated with Your exercise of permissions under this License.

   8. Limitation of Liability. In no event and under no legal theory,
      whether in tort (including negligence), contract, or otherwise,
      unless required by applicable law (such as deliberate and grossly
      negligent acts) or agreed to in writing, shall any Contributor be
      liable to You for damages, including any direct, indirect, special,
      incidental, or consequential damages of any character arising as a
      result of this License or out of the use or inability to use the
      Work (including but not limited to damages for loss of goodwill,
      work stoppage, computer failure or malfunction, or any and all
      other commercial damages or losses), even if such Contributor
      has been advised of the possibility of such damages.

   9. Accepting Warranty or Additional Liability. While redistributing
      the Work or Derivative Works thereof, You may choose to offer,
      and charge a fee for, acceptance of support, warranty, indemnity,
      or other liability obligations and/or rights consistent with this
      License. However, in accepting such obligations, You may act only
      on Your own behalf and on Your sole responsibility, not on behalf
      of any other Contributor, and only if You agree to indemnify,
      defend, and hold each Contributor harmless for any liability
      incurred by, or claims asserted against, such Contributor by reason
      of your accepting any such warranty or additional liability.

   END OF TERMS AND CONDITIONS

   APPENDIX: How to apply the Apache License to your work.

      To apply the Apache License to your work, attach the following
      boilerplate notice, with the fields enclosed by brackets "{}"
      replaced with your own identifying information. (Don't include
      the brackets!)  The text should be enclosed in the appropriate
      comment syntax for the file format. We also recommend that a
      file or class name and description of purpose be included on the
      same "printed page" as the copyright notice for easier
      identification within third-party archives.

   Copyright {yyyy} {name of copyright owner}

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
```

### `github.com/rivo/uniseg` — MIT License

```text
MIT License

Copyright (c) 2019 Oliver Kuederle

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### `github.com/jezek/xgb` — BSD-3-Clause with patent grant

```text
Copyright (c) 2009 The XGB Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

Subject to the terms and conditions of this License, Google hereby
grants to You a perpetual, worldwide, non-exclusive, no-charge,
royalty-free, irrevocable (except as stated in this section) patent
license to make, have made, use, offer to sell, sell, import, and
otherwise transfer this implementation of XGB, where such license
applies only to those patent claims licensable by Google that are
necessarily infringed by use of this implementation of XGB. If You
institute patent litigation against any entity (including a
cross-claim or counterclaim in a lawsuit) alleging that this
implementation of XGB or a Contribution incorporated within this
implementation of XGB constitutes direct or contributory patent
infringement, then any patent licenses granted to You under this
License for this implementation of XGB shall terminate as of the date
such litigation is filed.
```

### `golang.org/x/image`, `x/sync`, `x/sys`, and `x/text` — BSD-3-Clause

```text
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### `github.com/go-text/typesetting` — Unlicense OR BSD-3-Clause

```text
This project is provided under the terms of the UNLICENSE or
the BSD license denoted by the following SPDX identifier:

SPDX-License-Identifier: Unlicense OR BSD-3-Clause

You may use the project under the terms of either license.

Both licenses are reproduced below.

----
The BSD 3 Clause License

Copyright 2021 The go-text authors

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
---



---
The UNLICENSE

This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.

For more information, please refer to <https://unlicense.org/>
---
```

## Bundled Go Regular font

The UI embeds Go Regular from `golang.org/x/image/font/gofont/goregular`. Its complete license is
reproduced below from `golang.org/x/image/font/gofont/ttfs/README` at `v0.43.0`.

```text
These fonts were created by the Bigelow & Holmes foundry specifically for the
Go project. See https://blog.golang.org/go-fonts for details.

They are licensed under the same open source license as the rest of the Go
project's software:

Copyright (c) 2016 Bigelow & Holmes Inc.. All rights reserved.

Distribution of this font is governed by the following license. If you do not
agree to this license, including the disclaimer, do not distribute or modify
this font.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

    * Redistributions of source code must retain the above copyright notice,
      this list of conditions and the following disclaimer.

    * Redistributions in binary form must reproduce the above copyright notice,
      this list of conditions and the following disclaimer in the documentation
      and/or other materials provided with the distribution.

    * Neither the name of Google Inc. nor the names of its contributors may be
      used to endorse or promote products derived from this software without
      specific prior written permission.

DISCLAIMER: THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

## Development and release tooling

These packages and tools verify or optimize releases but are not shipped as part of the game
executable or WebAssembly module.

| Ecosystem | Component | Version | License | Current scope | Upstream record |
| --- | --- | --- | --- | --- | --- |
| npm | `fsevents` | `2.3.2` | MIT | Optional macOS browser-test dependency | [source](https://github.com/fsevents/fsevents/blob/v2.3.2/LICENSE) |
| npm | `playwright` | `1.62.1` | Apache-2.0 | Browser verification | [source](https://github.com/microsoft/playwright/blob/v1.62.1/LICENSE) |
| npm | `playwright-core` | `1.62.1` | Apache-2.0 | Browser verification | [source](https://github.com/microsoft/playwright/blob/v1.62.1/LICENSE) |
| Tool | `Binaryen` | `132` | Apache-2.0 | WebAssembly release optimization | [source](https://github.com/WebAssembly/binaryen/blob/version_132/LICENSE) |

GitHub Actions are SHA-pinned workflow services rather than modules or files redistributed with the
game. `golangci-lint` is installed at a pinned version in CI and release verification; its own module
graph is likewise not part of the application build list or a release archive.
