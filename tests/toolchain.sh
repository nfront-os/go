#!/bin/bash
# The installed toolchain (scripts/install.sh, out/$objtype) in the 9front guest, on an ext4
# disk at /$objtype/go: go version, go run, go build, gofmt, go vet, and go test for a few
# std packages (PKGS). The first go command compiles std in the guest (minutes).
# usage: [objtype=386|arm64] [PKGS='...'] tests/toolchain.sh
set -e
TOP=$(cd "$(dirname "$0")/.." && pwd)
A=${objtype:-amd64}
D=$TOP/out/$A/$A/go
[ -x "$D/bin/go" ] || { echo "toolchain.sh: no $D: objtype=$A scripts/install.sh" >&2; exit 1; }
PKGS=${PKGS:-errors strings strconv bytes sort unicode/utf8 fmt path/filepath os os/exec time sync net}
log=$(objtype=$A MEM=${MEM:-2048} GUEST_TIMEOUT=${GUEST_TIMEOUT:-5400} CMDWAIT=${CMDWAIT:-3000} \
	"$TOP/tests/guest.sh" -D "/$A/go=$D" "$TOP/tests/prog/hello.go" "$TOP/tests/prog/goroutines.go" -- \
	"ip/ipconfig loopback /dev/null 127.1; ip/ipconfig loopback /dev/null ::1 /128" \
	"bind -b /$A/go/bin /bin" \
	"cd /tmp; cp /tests/*.go .; go version" \
	"go run hello.go" \
	"go build -o g goroutines.go && ./g" \
	"echo 'package main;func main(){println(1)}' > f.go; gofmt f.go | wc -l | sed 's/^ */gofmt lines /'" \
	"go vet hello.go && echo vet ok" \
	"go test -short $PKGS >[2=1] | sed 's/^/T /'; echo test status \$status" 2>&1 | tee /dev/stderr)
# the guest did not come up (boot-cli.sh refused, QEMU failed): say why
grep -q '^init: starting' <<<"$log" || { echo "$log" | tail -20; echo "$(basename "$0"): the guest did not boot" >&2; exit 1; }
pass=0 fail=0
check() { if grep -qE "$2" <<<"$log"; then pass=$((pass+1)); echo "PASS $1"; else fail=$((fail+1)); echo "FAIL $1"; fi; }
check version "^go version go[0-9.]+ plan9/$A"
check run "hello from Go .* on plan9/$A"
check build "^ok goroutines"
check gofmt "gofmt lines +3"
check vet "^vet ok"
for p in $PKGS; do check "test $p" "^T ok[[:space:]]+$p[[:space:]]"; done
echo "$A toolchain: $pass/$((pass+fail)) passed"
[ $fail = 0 ]
