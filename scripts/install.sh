#!/bin/bash
# Go for an installed 9front system: the tree out/$objtype/$objtype/go that goes to
# /$objtype/go, and an archive of it with paths from / (on 9front: cd /; tar xzf
# out/$objtype/go-plan9-$objtype.tgz). ../9front-gccwt's scripts/guest/stage-dist.sh puts
# the tree on the install CD if asked (GO_OUT=out/$objtype).
#   bin/go, bin/gofmt            cmd/go and gofmt for plan9/$objtype
#   pkg/tool/plan9_$objtype/     compile, link, asm, vet, ...
#   pkg/include/                 the assembler's headers
#   src/, lib/, misc/            the sources of std and cmd (go/'s tracked files and
#                                what cmd/dist generated)
#   VERSION, go.env, LICENSE, PATENTS, README
# The commands are linked anew here with -trimpath and runtime.defaultGOROOT=/$objtype/go:
# go finds its tree with no $GOROOT, also when run as /bin/go (bind -b /$objtype/go/bin /bin).
# Needs scripts/build.sh first (the host toolchain in go/bin).
# usage: [objtype=386|arm64] scripts/install.sh [outdir]   (default out/$objtype)
set -e
TOP=$(cd "$(dirname "$0")/.." && pwd)
A=${objtype:-amd64}
OUT=${1:-$TOP/out/$A}
D=$OUT/$A/go
G=$TOP/go/bin/go
die() { echo "install.sh: $*" >&2; exit 1; }
[ -x "$G" ] || die "no $G: scripts/build.sh"

rm -rf "$D"
mkdir -p "$D/bin" "$D/pkg/tool/plan9_$A"
# the sources: go/'s tracked files (no build output)
(cd "$TOP/go" && git ls-files -z -- src lib misc api go.env VERSION LICENSE PATENTS |
	tar -c --null -T - -f -) | tar -x -C "$D" -f -
# and what cmd/dist generated (zversion.go, zbootstrap.go, zdefaultcc.go, the tzdata's
# zzipdata.go), as Go's own distributions have them
(cd "$TOP/go" && git ls-files -z -o -i --exclude-standard -- src | grep -z '\.go$' |
	tar -c --null -T - -f -) | tar -x -C "$D" -f -
# the assembler's headers (textflag.h, funcdata.h, asm_ppc64x.h), from cmd/dist
cp -r "$TOP/go/pkg/include" "$D/pkg/"
export GOOS=plan9 GOARCH=$A CGO_ENABLED=0 GOFLAGS=
flags=(-trimpath "-ldflags=-X=runtime.defaultGOROOT=/$A/go")
cd "$TOP/go/src"
"$G" build "${flags[@]}" -o "$D/bin/" cmd/go cmd/gofmt
tools=$(cd "$TOP/go/pkg/tool/plan9_$A" 2>/dev/null && ls) ||
	die "no go/pkg/tool/plan9_$A: objtype=$A scripts/build.sh"
for t in $tools; do
	"$G" build "${flags[@]}" -o "$D/pkg/tool/plan9_$A/$t" "cmd/$t"
done
version=$(head -1 "$TOP/go/VERSION")
port=$(cd "$TOP/go" && git log -1 --format='%h %cd' --date=short)
sed "s/@objtype@/$A/g; s/@version@/$version/; s/@port@/$port/" > "$D/README" <<'R'
Go for 9front (plan9/@objtype@), @version@, from 9front-go (port @port@)

	bind -b /@objtype@/go/bin /bin
	go run hello.go
	go build -o hello hello.go

bin/	go, gofmt
pkg/tool/plan9_@objtype@/	compile, link, asm, vet, ...
src/	std and cmd

go finds this tree as /@objtype@/go; set $GOROOT for another place. No cgo
(CGO_ENABLED=0). The build cache is in $home/lib/cache/go-build (GOCACHE),
modules in $home/go (GOPATH). go get reaches proxy.golang.org over https: it
needs the CA roots in /sys/lib/tls/ca.pem
R
cd "$OUT"
tar -czf "go-plan9-$A.tgz" "$A/go"
du -sh "$D" "go-plan9-$A.tgz"
