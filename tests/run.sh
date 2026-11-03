#!/bin/bash
# The programs of tests/prog, cross-built on the host for plan9/$objtype, run in the 9front
# guest (tests/guest.sh), SMP processors (4). Each prints "ok <name>" and no
# "FAIL <name>"; panic.go must die with a stack trace.
# usage: [objtype=386|arm64] tests/run.sh
set -e
TOP=$(cd "$(dirname "$0")/.." && pwd)
A=${objtype:-amd64}
B=$TOP/build/prog-$A
rm -rf "$B"; mkdir -p "$B"
progs=()
for f in "$TOP"/tests/prog/*.go; do
	n=$(basename "$f" .go)
	GOOS=plan9 GOARCH=$A CGO_ENABLED=0 "$TOP/go/bin/go" build -o "$B/$n" "$f"
	progs+=("$B/$n")
done
# processors of the guest (tests/guest.sh: SMP, 4), checked by parallel.go
smp=${SMP:-4}
# loopback (127.1) for net.go, as 9front-rust's tests do: the guest's boot sets up only ether0
cmds=("ip/ipconfig loopback /dev/null 127.1")
for p in "${progs[@]}"; do
	n=$(basename "$p")
	case $n in
	panic)	cmds+=("/tests/panic >[2=1]; echo panic status \$status") ;;
	parallel)	cmds+=("WANTCPU=$smp /tests/parallel") ;;
	*)	cmds+=("/tests/$n") ;;
	esac
done
# /tmp on an ext4 disk (tests/guest.sh -D): files.go and dirremove.go use ext4srv
mkdir -p "$TOP/build/empty"
log=$(objtype=$A SMP=$smp MEM=${MEM:-1024} "$TOP/tests/guest.sh" -D "/n/empty=$TOP/build/empty" "${progs[@]}" -- "${cmds[@]}" 2>&1 | tee /dev/stderr)
# the guest did not come up (boot-cli.sh refused, QEMU failed): say why
grep -q '^init: starting' <<<"$log" || { echo "$log" | tail -20; echo "$(basename "$0"): the guest did not boot" >&2; exit 1; }
pass=0 fail=0
for p in "${progs[@]}"; do
	n=$(basename "$p")
	case $n in
	panic)	ok=0; grep -q 'panic: gotest panic' <<<"$log" && grep -q 'main\.f' <<<"$log" &&
			grep -q "^panic status .*panic" <<<"$log" && ok=1 ;;
	hello)	ok=0; grep -q "hello from Go .* on plan9/$A" <<<"$log" && ok=1 ;;
	*)	ok=0; grep -q "^ok $n" <<<"$log" && ok=1 ;;
	esac
	! grep -q "^FAIL $n" <<<"$log" || ok=0
	if [ $ok = 1 ]; then pass=$((pass+1)); echo "PASS $n"; else fail=$((fail+1)); echo "FAIL $n"; fi
done
echo "$A: $pass/$((pass+fail)) passed"
[ $fail = 0 ]
