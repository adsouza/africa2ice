#!/bin/sh
# Verify that a tag is one the release workflow may build from: an annotated tag
# object whose commit is already reachable from origin/main.
#
# Both halves have a trap in them.
#
# The annotation check cannot be asked of the ref that actions/checkout leaves
# behind. Checkout fetches the real tag object first, but its third fetch is
# `+<commit-sha>:refs/tags/<tag>`, which force-writes the ref to point straight
# at the commit and discards the annotation. `git cat-file -t <tag>` after
# checkout therefore always answers `commit`, so an annotation check performed
# there can only ever fail. Origin still holds the tag object, so re-fetch the
# ref before asking.
#
# Reachability is checked against origin/main with an explicit refspec, because
# a tag checkout has no local main at all, and because relying on a remote's
# configured refspec to update refs/remotes/origin/main is how that ref ends up
# missing.
#
# Lint with `shellcheck`. Tested by tools/release_test.sh.

set -eu

if [ "$#" -ne 1 ]; then
	echo "usage: $0 TAG" >&2
	exit 2
fi
tag=$1

# Restore the annotated ref. Forced because checkout's rewrite is not a
# fast-forward; a no-op when the ref was never downgraded.
git fetch --force --quiet --no-tags origin "refs/tags/${tag}:refs/tags/${tag}"

object_type=$(git cat-file -t "$tag")
if [ "$object_type" != tag ]; then
	echo "check_release_tag: $tag is a $object_type, not an annotated tag object" >&2
	echo "check_release_tag: re-create it with git tag --annotate" >&2
	exit 1
fi

git fetch --quiet --no-tags origin "refs/heads/main:refs/remotes/origin/main"

commit=$(git rev-parse "${tag}^{commit}")
if ! git merge-base --is-ancestor "$commit" refs/remotes/origin/main; then
	echo "check_release_tag: $tag ($commit) is not reachable from origin/main" >&2
	exit 1
fi

echo "check_release_tag: $tag is annotated and $commit is on origin/main"
