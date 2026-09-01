#!/bin/sh
# Cut a release: check that this tree is releasable, push an annotated
# vMAJOR.MINOR.PATCH tag, and follow the Native release workflow it triggers
# until the GitHub release exists.
#
# Every check below mirrors a gate that .github/workflows/release.yml applies
# minutes later on a runner. Failing here costs a second and no version number;
# failing there abandons a tag that has already been published to origin. The
# accepted tag pattern must therefore stay identical to that workflow's push
# trigger, because a tag outside it pushes silently and builds nothing.
#
# Lint with `shellcheck`. Tested by tools/release_test.sh.

set -eu

watch=yes
if [ "${1:-}" = --no-watch ]; then
	watch=no
	shift
fi

if [ "$#" -ne 1 ]; then
	echo "usage: $0 [--no-watch] VERSION" >&2
	echo "example: $0 0.0.1" >&2
	exit 2
fi

version=$1
case "$version" in
	v*) ;;
	*) version="v$version" ;;
esac

# Identical to the tag filter in .github/workflows/release.yml.
if ! printf '%s\n' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
	echo "release: version must be vMAJOR.MINOR.PATCH, got $version" >&2
	exit 2
fi

branch=$(git rev-parse --abbrev-ref HEAD)
if [ "$branch" != main ]; then
	echo "release: releases are cut from main, but HEAD is on $branch" >&2
	exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
	echo "release: the working tree has uncommitted changes" >&2
	echo "release: commit or stash them, because the tag would not include them" >&2
	exit 1
fi

# An explicit refspec keeps refs/remotes/origin/main authoritative whatever the
# remote is configured to fetch, and --no-tags keeps an existing remote tag out
# of the local namespace so the two tag checks below stay distinguishable.
if ! git fetch --quiet --no-tags origin refs/heads/main:refs/remotes/origin/main; then
	echo "release: could not fetch main from origin" >&2
	exit 1
fi

head=$(git rev-parse HEAD)
upstream=$(git rev-parse refs/remotes/origin/main)
if [ "$head" != "$upstream" ]; then
	echo "release: HEAD is not origin/main, so the tag would fail the workflow's" >&2
	echo "release: reachable-from-main gate; push or pull first" >&2
	echo "release:   HEAD        $head" >&2
	echo "release:   origin/main $upstream" >&2
	exit 1
fi

if git rev-parse --verify --quiet "refs/tags/$version" >/dev/null; then
	echo "release: tag $version already exists locally" >&2
	exit 1
fi

if ! remote_tag=$(git ls-remote origin "refs/tags/$version"); then
	echo "release: could not list the tags on origin" >&2
	exit 1
fi
if [ -n "$remote_tag" ]; then
	echo "release: tag $version already exists on origin" >&2
	exit 1
fi

# The workflow requires a tag object rather than a lightweight ref, so annotate.
git tag --annotate "$version" --message "Africa 2 Ice $version"
if ! git push --quiet origin "refs/tags/$version"; then
	git tag --delete "$version" >/dev/null
	echo "release: pushing $version failed; removed the local tag so that" >&2
	echo "release: rerunning this command can retry the same version" >&2
	exit 1
fi
echo "release: pushed $version at $head"

if [ "$watch" = no ]; then
	echo "release: not following the workflow; watch it with 'gh run watch'"
	exit 0
fi

if ! command -v gh >/dev/null 2>&1; then
	echo "release: gh is not installed, so the workflow runs unobserved" >&2
	exit 0
fi

# A tag push registers its run asynchronously, so the run may not be listable
# for a few seconds after the push returns.
run=""
attempt=0
while [ "$attempt" -lt 20 ]; do
	run=$(gh run list --workflow release.yml --event push --branch "$version" \
		--limit 1 --json databaseId --jq '.[0].databaseId' 2>/dev/null || true)
	if [ -n "$run" ]; then
		break
	fi
	attempt=$((attempt + 1))
	sleep 3
done

if [ -z "$run" ]; then
	echo "release: $version is pushed, but no workflow run appeared in 60s" >&2
	echo "release: check the Actions tab before assuming the release failed" >&2
	exit 1
fi

echo "release: following run $run"
gh run watch "$run" --exit-status

echo "release: published $(gh release view "$version" --json url --jq .url)"
