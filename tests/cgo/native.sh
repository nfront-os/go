#!/bin/bash
# cgo on 9front itself: the installed Go (scripts/install.sh, out/$objtype) and 9front-gcc's
# native gcc (GCC_OUT, ../9front-gcc/out/$objtype: /$objtype/gnu) in the guest build
# and run the programs of tests/cgo with go build (CGO_ENABLED=1). The guest's root has no
# headers and archives: the tree's (PLAN9, ../9front-gccwt) go along on the ext4 disk.
# usage: [objtype=386|arm64] tests/cgo/native.sh
set -e
TOP=$(cd "$(dirname "$0")/../.." && pwd)
. "$TOP/scripts/nfront-paths.sh"	# TREE, TREE_GCC, GCC_DIR, ... (the distribution's paths.env)
A=${objtype:-amd64}
P=$(cd "${PLAN9:-$TREE_GCC}" && pwd)
G=${GCC_OUT:-$GCC_DIR/out/$A}/$A/gnu
D=$TOP/out/$A/$A/go
[ -x "$D/bin/go" ] || { echo "cgo/native.sh: no $D: objtype=$A scripts/install.sh" >&2; exit 1; }
[ -x "$G/bin/gcc" ] || { echo "cgo/native.sh: no $G/bin/gcc (GCC_DIR: toolchain/gcc, build.sh native)" >&2; exit 1; }
progs=() cmds=()
for f in "$TOP"/tests/cgo/*.go; do
	n=$(basename "$f" .go)
	progs+=("$f")
	cmds+=("CGO_ENABLED=1 go build -o $n $n.go >[2=1] && ./$n")
done
log=$(objtype=$A SMP=${SMP:-4} MEM=${MEM:-2048} GUEST_TIMEOUT=${GUEST_TIMEOUT:-5400} CMDWAIT=${CMDWAIT:-3000} \
	"$TOP/tests/guest.sh" -D "/$A/go=$D" -D "/$A/gnu=$G" -D "/sys/include=$P/sys/include" \
	-D "/$A/include=$P/$A/include" -D "/$A/lib.gcc=$P/$A/lib.gcc" -x "/$A/plan9.ld=$P/$A/plan9.ld" \
	"${progs[@]}" -- "bind -b /$A/gnu/bin /bin; bind -b /$A/go/bin /bin; cd /tmp; cp /tests/*.go ." \
	"${cmds[@]}" 2>&1 | tee /dev/stderr)
grep -q '^init: starting' <<<"$log" || { echo "$log" | tail -20; echo "cgo/native.sh: the guest did not boot" >&2; exit 1; }
pass=0 fail=0
for f in "${progs[@]}"; do
	n=$(basename "$f" .go)
	ok=0; grep -q "^ok $n" <<<"$log" && ok=1
	! grep -q "^FAIL $n" <<<"$log" || ok=0
	if [ $ok = 1 ]; then pass=$((pass+1)); echo "PASS $n"; else fail=$((fail+1)); echo "FAIL $n"; fi
done
echo "$A cgo native: $pass/$((pass+fail)) passed"
[ $fail = 0 ]
