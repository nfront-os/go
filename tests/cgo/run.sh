#!/bin/bash
# cgo: the programs of tests/cgo, cross-built on the host with 9front-gcc's cross gcc
# (GCC9BUILD, ../9front-gcc/build-gcc; the cgo program is an ELF linked against APE),
# run in the 9front guest (tests/guest.sh, SMP processors, 4). Each prints "ok <name>"
# and no "FAIL <name>".
# usage: [objtype=386|arm64] tests/cgo/run.sh
set -e
TOP=$(cd "$(dirname "$0")/../.." && pwd)
A=${objtype:-amd64}
GCC9BUILD=${GCC9BUILD:-$TOP/../9front-gcc/build-gcc}
case $A in
amd64)	X=x86_64-pc-plan9 ;;
386)	X=i686-pc-plan9 ;;
arm64)	X=aarch64-pc-plan9 ;;
*)	echo "cgo/run.sh: objtype=$A: no cross gcc" >&2; exit 1 ;;
esac
CC=$GCC9BUILD/$A/cross/bin/$X-gcc
[ -x "$CC" ] || { echo "cgo/run.sh: no $CC (../9front-gcc: build.sh cross)" >&2; exit 1; }
export GOCACHE=$TOP/build/gocache	# scripts/build.sh's
B=$TOP/build/cgo-$A
rm -rf "$B"; mkdir -p "$B"
progs=() cmds=()
for f in "$TOP"/tests/cgo/*.go; do
	n=$(basename "$f" .go)
	GOOS=plan9 GOARCH=$A CGO_ENABLED=1 CC=$CC "$TOP/go/bin/go" build -o "$B/$n" "$f"
	progs+=("$B/$n")
	cmds+=("/tests/$n")
done
# no proc of a program may be left (a thread of C's own after main returned)
names=$(cd "$B" && ls | tr '\n' '|' | sed 's/|$//')
cmds+=("sleep 2; ps | grep '$names' | sed 's/^/left /'; echo procs-checked")
log=$(objtype=$A SMP=${SMP:-4} MEM=${MEM:-1024} "$TOP/tests/guest.sh" "${progs[@]}" -- "${cmds[@]}" 2>&1 | tee /dev/stderr)
grep -q '^init: starting' <<<"$log" || { echo "$log" | tail -20; echo "cgo/run.sh: the guest did not boot" >&2; exit 1; }
pass=0 fail=0
for p in "${progs[@]}"; do
	n=$(basename "$p")
	ok=0; grep -q "^ok $n" <<<"$log" && ok=1
	! grep -q "^FAIL $n" <<<"$log" || ok=0
	if [ $ok = 1 ]; then pass=$((pass+1)); echo "PASS $n"; else fail=$((fail+1)); echo "FAIL $n"; fi
done
grep '^left ' <<<"$log" || true
if grep -q '^procs-checked' <<<"$log" && ! grep -q '^left ' <<<"$log"; then pass=$((pass+1)); echo "PASS no procs left"; else fail=$((fail+1)); echo "FAIL procs left after the programs"; fi
echo "$A cgo: $pass/$((pass+fail)) passed"
[ $fail = 0 ]
