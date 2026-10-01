#!/usr/bin/env bash
# Cross-compile release archives for the fork: linux/amd64 and windows/amd64.
#
#   scripts/fork-dist.sh                          # version from git describe
#   VERSION=v0.97.1-nebuloss.1 scripts/fork-dist.sh
#
# CGO is disabled, so one Linux machine builds every target with the stock Go
# toolchain. Output goes to dist/: one archive per target and a checksums file.
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
rm -rf dist build
mkdir -p dist

for target in linux/amd64 windows/amd64; do
	os=${target%/*}
	name=crush_${version}_${os}_x86_64
	ext=
	[ "$os" = windows ] && ext=.exe
	echo "==> $name"
	mkdir -p "build/$name"
	GOOS=$os GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags "$ldflags" \
		-o "build/$name/crush$ext" .
	cp LICENSE.md README.md FORK.md "build/$name/"
	if [ "$os" = windows ]; then
		(cd build && zip -qr "../dist/$name.zip" "$name")
	else
		tar -C build -czf "dist/$name.tar.gz" "$name"
	fi
done

(cd dist && sha256sum -- * > "crush_${version}_checksums.txt")
rm -rf build
ls -l dist
