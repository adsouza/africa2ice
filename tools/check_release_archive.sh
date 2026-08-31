#!/bin/sh
# Extract one native release archive and enforce its exact four-file contract.
# When an expected VCS revision is supplied, also run the extracted executable
# and verify that both its Go build metadata and session log identify that
# revision.

set -eu

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
	echo "usage: $0 ARCHIVE EXECUTABLE [EXPECTED_VCS_REVISION]" >&2
	exit 2
fi

archive=$1
executable=$2
expected_revision=${3:-}

test -f "$archive" || {
	echo "check_release_archive: archive not found: $archive" >&2
	exit 2
}

archive_file=${archive##*/}
case "$archive_file" in
	*.zip)
		root_name=${archive_file%.zip}
		format=zip
		;;
	*.tar.gz)
		root_name=${archive_file%.tar.gz}
		format=tar
		;;
	*)
		echo "check_release_archive: unsupported archive: $archive" >&2
		exit 2
		;;
esac

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/africa2ice-release-check.XXXXXX")
cleanup() {
	rm -rf "$work_dir"
}
trap cleanup EXIT HUP INT TERM

if [ "$format" = zip ]; then
	unzip -q "$archive" -d "$work_dir"
else
	tar -xzf "$archive" -C "$work_dir"
fi

root="$work_dir/$root_name"
test -d "$root" || {
	echo "check_release_archive: expected root directory $root_name" >&2
	exit 1
}

for required in "$executable" LICENSE THIRD_PARTY_NOTICES.md RUNNING.md; do
	path="$root/$required"
	if [ ! -f "$path" ] || [ -L "$path" ]; then
		echo "check_release_archive: missing or non-regular entry $required" >&2
		exit 1
	fi
done

# One root directory plus exactly four regular files; this also rejects hidden
# files, nested directories, alternate roots, and payload outside the root.
entry_count=$(find "$work_dir" -mindepth 1 -print | wc -l | tr -d '[:space:]')
if [ "$entry_count" -ne 5 ]; then
	echo "check_release_archive: expected one root plus four files, found $entry_count entries" >&2
	find "$work_dir" -mindepth 1 -print >&2
	exit 1
fi

if [ -n "$expected_revision" ]; then
	program="$root/$executable"
	go version -m "$program" | grep -Fq "vcs.revision=$expected_revision" || {
		echo "check_release_archive: executable build metadata does not identify $expected_revision" >&2
		exit 1
	}

	smoke_output="$work_dir/native-smoke.log"
	"$program" -headless -turns 0 >"$smoke_output"
	grep -q '^Africa 2 Ice session log:' "$smoke_output" || {
		echo "check_release_archive: executable did not announce a session log" >&2
		exit 1
	}
	session_log=$(sed -n 's/^Africa 2 Ice session log: //p' "$smoke_output" | head -n 1 | tr -d '\r')
	case "$(uname -s)" in
		MINGW* | MSYS* | CYGWIN*) session_log=$(cygpath -u "$session_log") ;;
	esac
	test -s "$session_log" || {
		echo "check_release_archive: announced session log is missing or empty" >&2
		exit 1
	}
	grep -Fq "\"vcs_revision\":\"$expected_revision\"" "$session_log" || {
		echo "check_release_archive: session.start does not identify $expected_revision" >&2
		exit 1
	}
fi

echo "release archive verified: $archive_file"
