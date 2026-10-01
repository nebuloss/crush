#!/usr/bin/env bash
# Rebase the fork's `patched` branch onto an upstream release.
#
#   scripts/fork-rebase.sh            # latest upstream release tag
#   scripts/fork-rebase.sh v0.98.0    # a specific one
#
# The fork deletes upstream's CI workflows. When upstream edits one of them,
# the rebase stops on a modify/delete conflict; those are resolved here
# automatically by keeping the deletion. Any other conflict stops the script
# with the rebase in progress; resolve and `git add` the files, then re-run
# this script to continue.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

in_rebase() { [ -d "$(git rev-parse --git-path rebase-merge)" ]; }
resolve() {
	# Keep our deletions of upstream files that upstream has since modified.
	# In a rebase "them" is the commit being replayed, hence UD.
	git status --porcelain | awk '$1 == "UD" { print $2 }' | xargs -r git rm -q
	git diff --name-only --diff-filter=U | grep -q . && return 1
	# Non-zero just means the next commit stopped too; the loop re-checks.
	GIT_EDITOR=true git rebase --continue || true
}
finish() {
	local steps=0
	while in_rebase; do
		if ((steps++ > 200)); then
			echo "rebase is not making progress; see git status" >&2
			exit 1
		fi
		resolve || {
			echo "conflicts need a human: fix them, git add, then re-run" >&2
			git diff --name-only --diff-filter=U >&2
			exit 1
		}
	done
	git log --oneline "$(git describe --tags --abbrev=0 --exclude '*-*' patched)"..patched
	echo "==> done; test, then: git push --force-with-lease origin patched"
}

# Re-run after fixing a conflict by hand: carry on with the rebase.
if in_rebase; then
	finish
	exit 0
fi
git remote get-url upstream >/dev/null 2>&1 ||
	git remote add upstream https://github.com/charmbracelet/crush.git
# --force: upstream moves its `nightly` tag daily.
git fetch -q --tags --force upstream

release_tags() { git tag -l 'v[0-9]*.[0-9]*.[0-9]*' | grep -v -- - | sort -V; }

target=${1:-$(release_tags | tail -1)}
# Our base is the newest plain release tag that is an ancestor of patched.
base=$(for t in $(release_tags | tac); do
	git merge-base --is-ancestor "$t" patched && { echo "$t"; break; }
done)
[ -n "$base" ] || { echo "cannot find the release patched is based on" >&2; exit 1; }

if [ "$base" = "$target" ]; then
	echo "patched is already on $target"
	exit 0
fi

echo "==> rebasing patched from $base onto $target"
git checkout -q patched
git rebase -q --onto "$target" "$base" patched || true
finish
