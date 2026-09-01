#!/bin/sh
# Install the pinned golangci-lint release binary into the directory given as
# the first argument.
#
# `go install .../golangci-lint@vX.Y.Z` compiles the linter from source on every
# run. Measured across three CI runs that costs 55-59 s on the Ubuntu runners
# every time, in both workflows, to produce a binary the project already
# publishes. Fetching the published one takes seconds.
#
# This mirrors the pinned-download-plus-checksum pattern the workflows already
# use for Binaryen: the version and every digest live here, so a substituted or
# corrupted download fails closed instead of silently linting with a different
# tool than the one the repository pinned. Bumping the version means replacing
# the digests too, which is the point -- an unverifiable bump cannot pass.
#
# Digests are the upstream release's published
# golangci-lint-2.12.2-checksums.txt entries.
#
# Lint with `shellcheck`.

set -eu

version=2.12.2

if [ "$#" -ne 1 ]; then
	echo "usage: $0 DESTINATION_DIRECTORY" >&2
	exit 2
fi
destination=$1

case "$(uname -s)" in
	Linux) platform=linux ;;
	Darwin) platform=darwin ;;
	MINGW* | MSYS* | CYGWIN*) platform=windows ;;
	*)
		echo "install_golangci_lint: unsupported system $(uname -s)" >&2
		exit 1
		;;
esac

case "$(uname -m)" in
	x86_64 | amd64) architecture=amd64 ;;
	arm64 | aarch64) architecture=arm64 ;;
	*)
		echo "install_golangci_lint: unsupported architecture $(uname -m)" >&2
		exit 1
		;;
esac

# Only the three targets the CI and release matrices actually run are pinned; a
# fourth would need its own reviewed digest rather than a silent fallback.
case "$platform-$architecture" in
	linux-amd64)
		archive="golangci-lint-$version-linux-amd64.tar.gz"
		digest=8df580d2670fed8fa984aac0507099af8df275e665215f5c7a2ae3943893a553
		;;
	darwin-arm64)
		archive="golangci-lint-$version-darwin-arm64.tar.gz"
		digest=a9c54498731b3128f79e090be6110f3e5fffccc617b08142ed244d4126c73f29
		;;
	windows-amd64)
		archive="golangci-lint-$version-windows-amd64.zip"
		digest=bd42e3ebc8cb4ececb86941983baaf1dc221bbb04d838e94ce63b49cc91e02bb
		;;
	*)
		echo "install_golangci_lint: no pinned digest for $platform-$architecture" >&2
		exit 1
		;;
esac

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/golangci-lint-install.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

curl -L --fail --silent --show-error \
	--output "$work_dir/$archive" \
	"https://github.com/golangci/golangci-lint/releases/download/v$version/$archive"

# Runners disagree about which checksum tool exists, and a name existing does
# not mean the interface does: macOS has a sha256sum with no --check at all. So
# compute the digest and compare it here rather than delegating the comparison.
digest_of() {
	for tool in "sha256sum" "shasum -a 256"; do
		# Intentional word splitting: $tool carries its arguments.
		# shellcheck disable=SC2086
		if value=$($tool "$1" 2>/dev/null | awk '{print $1}') &&
			[ "${#value}" -eq 64 ]; then
			printf '%s\n' "$value"
			return 0
		fi
	done
	return 1
}

measured=$(digest_of "$work_dir/$archive") || {
	echo "install_golangci_lint: no working sha256 tool to verify $archive" >&2
	exit 1
}
if [ "$measured" != "$digest" ]; then
	echo "install_golangci_lint: $archive has digest" >&2
	echo "install_golangci_lint:   $measured" >&2
	echo "install_golangci_lint: but the pinned digest is" >&2
	echo "install_golangci_lint:   $digest" >&2
	exit 1
fi

case "$archive" in
	*.tar.gz) tar -xzf "$work_dir/$archive" -C "$work_dir" ;;
	*.zip)
		# GitHub's Windows images ship 7-Zip but no unzip.
		if command -v unzip >/dev/null 2>&1; then
			unzip -q "$work_dir/$archive" -d "$work_dir"
		elif sevenzip=$(command -v 7z || command -v 7zz); then
			"$sevenzip" x -bso0 -bsp0 -o"$work_dir" "$work_dir/$archive"
		else
			echo "install_golangci_lint: no unzip and no 7-Zip to extract $archive" >&2
			exit 1
		fi
		;;
esac

executable=golangci-lint
if [ "$platform" = windows ]; then
	executable=golangci-lint.exe
fi
extracted="$work_dir/golangci-lint-$version-$platform-$architecture/$executable"
test -f "$extracted" || {
	echo "install_golangci_lint: $archive did not contain $executable" >&2
	exit 1
}

mkdir -p "$destination"
cp "$extracted" "$destination/$executable"
chmod +x "$destination/$executable"

# Prove the installed binary runs and is the pinned version, rather than
# trusting that a verified archive produced a working tool.
installed=$("$destination/$executable" --version)
case "$installed" in
	*"$version"*) ;;
	*)
		echo "install_golangci_lint: installed binary reports \"$installed\", wanted $version" >&2
		exit 1
		;;
esac
echo "install_golangci_lint: $installed"
