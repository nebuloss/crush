#!/usr/bin/env bash
# Cross-compile release binaries for the fork: linux/amd64 and windows/amd64.
#
#   scripts/fork-dist.sh                          # version from git describe
#   VERSION=v0.97.1-nebuloss.1 scripts/fork-dist.sh
#
# CGO is disabled, so one Linux machine builds every target with the stock Go
# toolchain. Output goes to dist/: one self-contained binary per target, named
# crush-<os>-<arch>[.exe] (the release tag carries the version, so the
# latest-release download URL is stable), plus SHA256SUMS.
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
	arch=${target#*/}
	ext=
	[ "$os" = windows ] && ext=.exe
	echo "==> crush-$os-$arch$ext"
	GOOS=$os GOARCH=$arch go build -trimpath -buildvcs=false -ldflags "$ldflags" \
		-o "dist/crush-$os-$arch$ext" .
done

(cd dist && sha256sum crush-* > SHA256SUMS)
ls -l dist
