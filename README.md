# 9front-go

Go for 9front (Plan 9): `plan9/amd64`, `plan9/386` (Go's own ports) and `plan9/arm64`
(new here: Go has no Plan 9 port for arm64), cross-compiled on Linux, for the gcc-built
9front tree, branch `gcc` (`../9front-gccwt`). Programs are a.out, as from 6l/8l/7l, and
run on 9front as well.

Log of the port (German): `NOTES.md`.

## Layout

- `go/`: the Go sources (go1.27.3), its own git repository; the port is on its branch
  `plan9` (`git -C go log go1.27.3..plan9`).
- `scripts/build.sh`: the host toolchain (`go/src/make.bash`, bootstrapped with the host's
  Go), then std and cmd for `GOOS=plan9 GOARCH=$objtype`.
- `scripts/install.sh`: the tree for an installed system, `out/$objtype/$objtype/go`
  (`/$objtype/go`), and `out/$objtype/go-plan9-$objtype.tgz`.
- `tests/`: `guest.sh` runs programs in the 9front guest (from 9front-rust),
  `run.sh` the programs of `tests/prog` (cross-built), `toolchain.sh` the installed
  toolchain in the guest (go run, go build, gofmt, go vet, go test).

## Build

    git clone --branch plan9 <this go> go   # or: the go1.27.3 tag and the commits of plan9
    objtype=amd64 scripts/build.sh           # 386, arm64: HOST=0 skips the host toolchain
    objtype=amd64 scripts/install.sh

## Use

Cross-compiling:

    GOOS=plan9 GOARCH=arm64 go/bin/go build -o hello hello.go

On 9front (`cd /; tar xzf go-plan9-amd64.tgz`, or the install CD with `GO_OUT`):

    bind -b /amd64/go/bin /bin
    go run hello.go

`go` finds its tree as `/$objtype/go` (linked with `runtime.defaultGOROOT`); no `$GOROOT`
needed. The first build compiles std in the guest (about two minutes on amd64 under KVM).

## Tests

    objtype=amd64 tests/run.sh          # 7 programs: goroutines, timers, files/exec, TCP, notes, panic
    objtype=arm64 tests/toolchain.sh    # the toolchain in the guest

To put the tree on the install CD: `GO_OUT=../9front-go/out/amd64` for
`../9front-gccwt`'s `scripts/guest/mkiso.sh`.
