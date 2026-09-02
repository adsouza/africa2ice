#!/bin/sh
# Exercise tools/release.sh against throwaway repositories.
#
# Every guard in release.sh stands between a mistake and an irreversible tag
# push, and a guard that quietly stops guarding looks exactly like a guard that
# works. So each case here builds a fresh bare "origin" plus a working clone,
# moves it into one bad state, and asserts release.sh refuses with the message
# that names that state. The last cases assert the success path really does put
# an annotated tag on origin, because a preflight that rejects everything would
# also pass every rejection case.
#
# Fixtures are isolated from the caller's git configuration: a global
# commit.gpgsign, init.defaultBranch, or diff.external must not change results.
#
# Lint with `shellcheck`.

set -eu

tools_dir="$(cd "$(dirname "$0")" && pwd)"
release_script="$tools_dir/release.sh"
tag_script="$tools_dir/check_release_tag.sh"
for script in "$release_script" "$tag_script"; do
	test -x "$script" || {
		echo "release_test: $script is missing or not executable" >&2
		exit 2
	}
done

work_root="$(mktemp -d "${TMPDIR:-/tmp}/africa2ice-release-test.XXXXXX")"
trap 'rm -rf "$work_root"' EXIT HUP INT TERM
log="$work_root/release.log"

GIT_CONFIG_GLOBAL=/dev/null
GIT_CONFIG_SYSTEM=/dev/null
GIT_AUTHOR_NAME="Release Test"
GIT_AUTHOR_EMAIL=release-test@example.invalid
GIT_COMMITTER_NAME="$GIT_AUTHOR_NAME"
GIT_COMMITTER_EMAIL="$GIT_AUTHOR_EMAIL"
export GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM
export GIT_AUTHOR_NAME GIT_AUTHOR_EMAIL GIT_COMMITTER_NAME GIT_COMMITTER_EMAIL

cases=0
failures=0

fail() {
	failures=$((failures + 1))
	echo "FAIL $1" >&2
	if [ -s "$log" ]; then
		sed 's/^/     | /' "$log" >&2
	fi
}

# fixture NAME sets $work to a clone whose $origin bare repository holds one
# shared commit on main.
fixture() {
	origin="$work_root/$1/origin.git"
	work="$work_root/$1/work"
	mkdir -p "$work_root/$1"
	git init --quiet --bare -b main "$origin"
	git init --quiet -b main "$work"
	echo seed >"$work/seed.txt"
	git -C "$work" add seed.txt
	git -C "$work" commit --quiet --message Seed
	git -C "$work" remote add origin "$origin"
	git -C "$work" push --quiet origin main
}

# Both helpers force --no-watch. No test may poll GitHub, and a guard that
# stops guarding must fail fast here rather than fall through into release.sh's
# minute-long wait for a workflow run that a fixture will never produce.
#
# expect_refusal DESCRIPTION EXPECTED_MESSAGE ARGS... runs release.sh in $work
# and requires a non-zero exit whose output mentions EXPECTED_MESSAGE.
expect_refusal() {
	description=$1
	expected=$2
	shift 2
	cases=$((cases + 1))
	if (cd "$work" && "$release_script" --no-watch "$@") >"$log" 2>&1; then
		fail "$description: release.sh succeeded but should have refused"
		return 0
	fi
	if ! grep -Fq "$expected" "$log"; then
		fail "$description: refused without mentioning \"$expected\""
		return 0
	fi
	echo "ok   $description"
}

# expect_success DESCRIPTION ARGS... runs release.sh in $work and requires a
# zero exit.
expect_success() {
	description=$1
	shift
	cases=$((cases + 1))
	if ! (cd "$work" && "$release_script" --no-watch "$@") >"$log" 2>&1; then
		fail "$description: release.sh refused but should have succeeded"
		return 1
	fi
	echo "ok   $description"
}

# expect_equal DESCRIPTION EXPECTED ACTUAL
expect_equal() {
	cases=$((cases + 1))
	if [ "$2" != "$3" ]; then
		fail "$1: expected \"$2\", got \"$3\""
		return 0
	fi
	echo "ok   $1"
}

fixture usage
expect_refusal "rejects a missing version" "usage:"
expect_refusal "rejects a second argument" "usage:" 0.0.1 0.0.2
for malformed in 1.0 v1.2 1.2.3.4 v1.2.3-rc1 latest v0.0.1+build; do
	expect_refusal "rejects the malformed version $malformed" \
		"vMAJOR.MINOR.PATCH" "$malformed"
done

fixture branch
git -C "$work" checkout --quiet -b topic
expect_refusal "rejects a branch other than main" "cut from main" 0.0.1

fixture detached
git -C "$work" checkout --quiet --detach HEAD
expect_refusal "rejects a detached HEAD" "cut from main" 0.0.1

fixture unstaged
echo drift >>"$work/seed.txt"
expect_refusal "rejects an unstaged change" "uncommitted changes" 0.0.1

fixture staged
echo drift >>"$work/seed.txt"
git -C "$work" add seed.txt
expect_refusal "rejects a staged change" "uncommitted changes" 0.0.1

fixture ahead
echo later >"$work/later.txt"
git -C "$work" add later.txt
git -C "$work" commit --quiet --message Later
expect_refusal "rejects a commit that is not yet on origin/main" \
	"is not origin/main" 0.0.1

fixture behind
echo later >"$work/later.txt"
git -C "$work" add later.txt
git -C "$work" commit --quiet --message Later
git -C "$work" push --quiet origin main
git -C "$work" reset --quiet --hard HEAD~1
expect_refusal "rejects a stale main behind origin/main" "is not origin/main" 0.0.1

fixture localtag
git -C "$work" tag --annotate v0.0.1 --message Prior
expect_refusal "rejects a tag that already exists locally" \
	"already exists locally" 0.0.1

fixture remotetag
git -C "$origin" tag --annotate v0.0.1 --message Prior main
expect_refusal "rejects a tag that already exists on origin" \
	"already exists on origin" 0.0.1

fixture success
head="$(git -C "$work" rev-parse HEAD)"
if expect_success "pushes an unprefixed version as a v-prefixed tag" 0.0.1; then
	expect_equal "creates an annotated tag object" tag \
		"$(git -C "$work" cat-file -t v0.0.1)"
	expect_equal "points the tag at the released commit" "$head" \
		"$(git -C "$work" rev-parse 'v0.0.1^{commit}')"
	expect_equal "publishes the tag to origin" "$head" \
		"$(git -C "$origin" rev-parse 'refs/tags/v0.0.1^{commit}')"
	expect_equal "keeps the tag annotated on origin" tag \
		"$(git -C "$origin" cat-file -t refs/tags/v0.0.1)"
fi

fixture prefixed
expect_success "accepts an already v-prefixed version" v0.0.1
expect_equal "normalizes to a single v prefix" tag \
	"$(git -C "$work" cat-file -t v0.0.1)"

# A refused push must not leave the local tag behind, or the next attempt at the
# same version fails on the local-tag guard instead of retrying cleanly.
fixture unreachable
git -C "$work" remote set-url --push origin "$work_root/unreachable/absent.git"
expect_refusal "reports a failed push" "failed" 0.0.1
expect_equal "removes the local tag after a failed push" "" \
	"$(git -C "$work" tag --list v0.0.1)"

# --- tools/check_release_tag.sh ------------------------------------------------
#
# The workflow step this script replaces was reachable only by a real tag push,
# so no dry run could execute it -- and it was unconditionally broken for the
# whole time it sat there. The clobber case below is that regression: it replays
# the fetch sequence actions/checkout actually performs, which rewrites
# refs/tags/<tag> to point at the commit and destroys the annotation.

# tag_fixture NAME sets $origin (bare), $seed (a pusher) and $runner (a
# checkout-like clone with no local main).
tag_fixture() {
	origin="$work_root/$1/origin.git"
	seed="$work_root/$1/seed"
	runner="$work_root/$1/runner"
	mkdir -p "$work_root/$1"
	git init --quiet --bare -b main "$origin"
	git init --quiet -b main "$seed"
	echo seed >"$seed/f"
	git -C "$seed" add f
	git -C "$seed" commit --quiet --message Seed
	git -C "$seed" remote add origin "$origin"
	git -C "$seed" push --quiet origin main
	git init --quiet -b main "$runner"
	git -C "$runner" remote add origin "$origin"
}

# simulate_checkout TAG reproduces actions/checkout's three fetches, including
# the third one that downgrades the tag ref to a lightweight ref.
simulate_checkout() {
	git -C "$runner" fetch --quiet --prune origin \
		"+refs/heads/*:refs/remotes/origin/*" "+refs/tags/*:refs/tags/*"
	checkout_sha=$(git -C "$runner" rev-parse "refs/tags/$1^{commit}")
	git -C "$runner" fetch --quiet --no-tags --prune origin \
		"+${checkout_sha}:refs/tags/$1"
	git -C "$runner" checkout --quiet --force "refs/tags/$1"
}

# expect_tag_refusal DESCRIPTION EXPECTED_MESSAGE TAG
expect_tag_refusal() {
	cases=$((cases + 1))
	if (cd "$runner" && "$tag_script" "$3") >"$log" 2>&1; then
		fail "$1: check_release_tag.sh accepted a tag it should refuse"
		return 0
	fi
	if ! grep -Fq "$2" "$log"; then
		fail "$1: refused without mentioning \"$2\""
		return 0
	fi
	echo "ok   $1"
}

# expect_tag_success DESCRIPTION TAG
expect_tag_success() {
	cases=$((cases + 1))
	if ! (cd "$runner" && "$tag_script" "$2") >"$log" 2>&1; then
		fail "$1: check_release_tag.sh refused a tag it should accept"
		return 0
	fi
	echo "ok   $1"
}

tag_fixture annotated
git -C "$seed" tag --annotate v1.0.0 --message "Africa 2 Ice v1.0.0"
git -C "$seed" push --quiet origin refs/tags/v1.0.0
simulate_checkout v1.0.0
expect_equal "actions/checkout really does downgrade the tag ref" commit \
	"$(git -C "$runner" cat-file -t v1.0.0)"
expect_tag_success "accepts an annotated tag on main after checkout's rewrite" v1.0.0

tag_fixture lightweight
git -C "$seed" tag v1.0.0
git -C "$seed" push --quiet origin refs/tags/v1.0.0
simulate_checkout v1.0.0
expect_tag_refusal "refuses a lightweight tag" "not an annotated tag object" v1.0.0

tag_fixture offmain
git -C "$seed" checkout --quiet -b sidetrack
echo side >"$seed/side.txt"
git -C "$seed" add side.txt
git -C "$seed" commit --quiet --message Side
git -C "$seed" tag --annotate v1.0.0 --message "Africa 2 Ice v1.0.0"
git -C "$seed" push --quiet origin refs/tags/v1.0.0
simulate_checkout v1.0.0
expect_tag_refusal "refuses a tag whose commit is not on main" \
	"not reachable from origin/main" v1.0.0

tag_fixture absent
git -C "$runner" fetch --quiet origin "+refs/heads/*:refs/remotes/origin/*"
expect_tag_refusal "refuses a tag that origin does not have" "v9.9.9" v9.9.9

if [ "$failures" -ne 0 ]; then
	echo "release_test: $failures of $cases assertions failed" >&2
	exit 1
fi
echo "release_test: all $cases assertions passed"
