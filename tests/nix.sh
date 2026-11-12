#!/bin/bash
# Go on a NIX kernel (../9front-nix, branch nix: application cores, port/nix.c): boots it
# with *nixac=$NAC of $NCPU processors (default 1 of 4) and runs the programs of tests/prog
# twice, on the time-sharing cores as usual and through execac(1) on the last
# application core; tests/nix/core.go shows where the procs run (main on the AC, the
# procs Go makes on the TCs, as NIX starts a child of an AC process); tests/nix/acgo.go
# (on a TC only) moves one goroutine's proc to the AC itself with rfork(RFCORE).
# usage: [objtype=amd64|arm64] [PLAN9=../9front-nix] tests/nix.sh
set -e
TOP=$(cd "$(dirname "$0")/.." && pwd)
. "$TOP/scripts/nfront-paths.sh"	# TREE, TREE_GCC, GCC_DIR, ... (the distribution's paths.env)
A=${objtype:-amd64}
case $A in amd64|arm64) ;; *) echo "nix.sh: no application cores on $A" >&2; exit 1 ;; esac
export PLAN9=$(cd "${PLAN9:-$TREE_NIX}" && pwd)
NCPU=${NCPU:-4} NAC=${NAC:-1}
ac=$((NCPU-1))
B=$TOP/build/nix-$A
export GOCACHE=$TOP/build/gocache	# scripts/build.sh's
rm -rf "$B"; mkdir -p "$B"
progs=()
for f in "$TOP"/tests/prog/*.go "$TOP"/tests/nix/core.go "$TOP"/tests/nix/acgo.go; do
	n=$(basename "$f" .go)
	GOOS=plan9 GOARCH=$A CGO_ENABLED=0 "$TOP/go/bin/go" build -o "$B/$n" "$f"
	progs+=("$B/$n")
done
# loopback (127.1) for net.go; the AC's lines carry "ac: " (sed runs on a TC)
cmds=("ip/ipconfig loopback /dev/null 127.1" "echo acs \`{grep -c ' AC\$' /dev/sysstat}")
for p in "${progs[@]}"; do
	n=$(basename "$p")
	case $n in
	panic)	run="/tests/panic >[2=1]" ;;
	parallel)	run="WANTCPU=$NCPU /tests/parallel" ;;	# Go counts /dev/sysstat's lines, ACs too
	*)	run="/tests/$n" ;;
	esac
	[ $n = core ] || cmds+=("$run; echo $n status \$status")
	[ $n = acgo ] && continue	# takes the AC itself
	cmds+=("execac -c $ac /bin/rc -c '$run' | sed 's/^/ac: /'; echo ac $n status \$status")
done
mkdir -p "$TOP/build/empty"
# every program twice: more time than guest.sh's default (600 s)
log=$(objtype=$A SMP=$NCPU MEM=${MEM:-1024} GUEST_TIMEOUT=${GUEST_TIMEOUT:-1800} KARGS="${KARGS:+$KARGS }*ncpu=$NCPU *nixac=$NAC" \
	"$TOP/tests/guest.sh" -D "/n/empty=$TOP/build/empty" "${progs[@]}" -- "${cmds[@]}" 2>&1 | tr -d '\r' | tee /dev/stderr)
grep -q '^init: starting' <<<"$log" || { echo "$log" | tail -20; echo "nix.sh: the guest did not boot" >&2; exit 1; }
pass=0 fail=0
res(){ if [ $1 = 1 ]; then pass=$((pass+1)); echo "PASS $2"; else fail=$((fail+1)); echo "FAIL $2"; fi; }
ok=0; grep -qx "acs $NAC" <<<"$log" && ok=1; res $ok "$NAC application core(s) in /dev/sysstat"
for p in "${progs[@]}"; do
	n=$(basename "$p")
	for where in tc ac; do
		[ $where = tc ] && [ $n = core ] && continue
		[ $where = ac ] && [ $n = acgo ] && continue
		if [ $where = ac ]; then l=$(sed -n 's/^ac: //p' <<<"$log"); else l=$(grep -v '^ac: ' <<<"$log"); fi
		ok=0
		case $n in
		panic)	grep -q 'panic: gotest panic' <<<"$l" && grep -q 'main\.f' <<<"$l" && ok=1 ;;
		hello)	grep -q "hello from Go .* on plan9/$A" <<<"$l" && ok=1 ;;
		core)	grep -qx "core main $ac" <<<"$l" && ! grep "^core other" <<<"$l" | grep -qx "core other $ac" &&
				[ "$(grep -c '^core other' <<<"$l")" = 3 ] && grep -qx 'ok core' <<<"$l" && ok=1 ;;
		*)	grep -q "^ok $n" <<<"$l" && ok=1 ;;
		esac
		! grep -q "^FAIL $n" <<<"$l" || ok=0
		res $ok "$n ($where)"
	done
done
echo "$A nix: $pass/$((pass+fail)) passed"
[ $fail = 0 ]
