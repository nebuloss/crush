#!/usr/bin/env bash
# Cross-compile release binaries for the fork: linux/amd64 and windows/amd64.
#
#   scripts/fork-dist.sh                          # version from git describe
#   VERSION=v0.97.1-nebuloss.1 scripts/fork-dist.sh
#
# CGO is disabled, so one Linux machine builds every target with the stock Go
# toolchain. Output goes to dist/: one self-contained binary per target, as
# crush_<version>_<os>_x86_64[.exe]. GitHub shows each asset's SHA-256.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

VERSION=${VERSION:-$(git describe --tags --always)}
version=${VERSION#v}
commit=$(git rev-parse HEAD)
pkg=github.com/charmbracelet/crush/internal/version
# -buildvcs=false: otherwise Go stamps the module version from git, and
# internal/version prefers that over the -X value below.
ldflags="-s -w -X $pkg.Version=$version -X $pkg.Commit=$commit"

export CGO_ENABLED=0 GOEXPERIMENT=greenteagc
rm -rf dist
mkdir -p dist

for target in linux/amd64 windows/amd64; do
	os=${target%/*}
	name=crush_${version}_${os}_x86_64
	ext=
	[ "$os" = windows ] && ext=.exe
	echo "==> $name$ext"
	GOOS=$os GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags "$ldflags" \
		-o "dist/$name$ext" .
done

ls -l dist
