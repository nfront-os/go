#!/bin/bash
# Go for 9front, built on Linux: the host toolchain from go/ (src/make.bash, bootstrapped
# with the host's Go), then cmd and std for GOOS=plan9 GOARCH=$objtype with it
# (go/bin/plan9_$A, go/pkg/tool/plan9_$A). scripts/install.sh makes the tree for 9front.
# usage: [objtype=386|amd64|arm64] scripts/build.sh   (HOST=0: skip make.bash)
set -e
TOP=$(cd "$(dirname "$0")/.." && pwd)
A=${objtype:-amd64}
case $A in 386|amd64|arm64) ;; *) echo "build.sh: objtype=$A: no port" >&2; exit 1 ;; esac
export GOROOT_BOOTSTRAP=${GOROOT_BOOTSTRAP:-$(go env GOROOT)}
cd "$TOP/go/src"
# the host toolchain, if not there or if asked
if [ "${HOST:-1}" = 1 ] || [ ! -x "$TOP/go/bin/go" ]; then
	env -u GOROOT -u GOOS -u GOARCH ./make.bash
fi
G=$TOP/go/bin/go
GOOS=plan9 GOARCH=$A CGO_ENABLED=0 GOFLAGS= "$G" install std cmd
echo "build.sh: $TOP/go/bin/plan9_$A, $TOP/go/pkg/tool/plan9_$A"
