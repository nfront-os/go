# 9front-go

Go for 9front (Plan 9): `plan9/amd64`, `plan9/386` (Go's own ports) and `plan9/arm64`
(new here: Go has no Plan 9 port for arm64), cross-compiled on Linux, for the gcc-built
9front tree, branch `gcc` (`../9front-gccwt`). Programs are a.out, as from 6l/8l/7l, and
run on 9front as well.

Log of the port (German): `NOTES.md`.

## Layout

- `go/`: the Go sources (go1.27.3), its own git repository (ignored here); the port is on
  its branch `plan9` (`git -C go log go1.27.3..plan9`).
- `patches/go/`: that branch as patches (`scripts/patches.sh` writes them after a commit in
  `go/`); this repository holds the port through them.
- `scripts/build.sh`: the host toolchain (`go/src/make.bash`, bootstrapped with the host's
  Go), then std and cmd for `GOOS=plan9 GOARCH=$objtype`.
- `scripts/install.sh`: the tree for an installed system, `out/$objtype/$objtype/go`
  (`/$objtype/go`), and `out/$objtype/go-plan9-$objtype.tgz`.
- `tests/`: `guest.sh` runs programs in the 9front guest (from 9front-rust),
  `run.sh` the programs of `tests/prog` (cross-built), `toolchain.sh` the installed
  toolchain in the guest (go run, go build, gofmt, go vet, go test).

## Build

    git clone --depth 1 --branch go1.27.3 https://go.googlesource.com/go go
    git -C go switch -c plan9 && git -C go am $PWD/patches/go/*.patch
    objtype=amd64 scripts/build.sh           # 386, arm64: HOST=0 skips the host toolchain
    objtype=amd64 scripts/install.sh

## Use

Cross-compiling:

    GOOS=plan9 GOARCH=arm64 go/bin/go build -o hello hello.go

On 9front (`cd /; tar xzf go-plan9-amd64.tgz`, or the install CD with `GO_OUT`):

    bind -b /amd64/go/bin /bin
    go run hello.go

`go` finds its tree as `/$objtype/go` (linked with `runtime.defaultGOROOT`); no `$GOROOT`
needed. On the live CD `$home` is read-only: `GOCACHE=/tmp/go-build` for the build cache. The first build compiles std in the guest (about two minutes on amd64 under KVM).

## cgo (amd64)

cgo programs are linked by [9front-gcc](../9front-gcc)'s gcc against APE (POSIX C, pthreads)
into an ELF program, which the gcc port's kernel loads (without cgo: a.out, as before).
Cross:

    PATH=../9front-gcc/build-gcc/amd64/cross/bin:$PATH \
    CGO_ENABLED=1 CC=x86_64-pc-plan9-gcc GOOS=plan9 GOARCH=amd64 go/bin/go build prog.go

On 9front, with `/amd64/gnu` installed (`bind -b /amd64/gnu/bin /bin`): `go build` finds gcc
and builds cgo programs as any. C's errno comes as `syscall.Errno` with APE's numbers
(`syscall.Errno(C.ENOENT)`). In a cgo program the Go heap is a segment of its own and C's
malloc has the break; g is in APE's `_privates[1]`.

## Tests

    objtype=amd64 tests/run.sh          # 7 programs: goroutines, timers, files/exec, TCP, notes, panic
    objtype=arm64 tests/toolchain.sh    # the toolchain in the guest
    tests/cgo/run.sh                    # cgo, cross-built: C calls Go, C threads in Go, notes, errno
    tests/cgo/native.sh                 # the same, built by go and gcc in the guest

To put the tree on the install CD: `GO_OUT=../9front-go/out/amd64` for
`../9front-gccwt`'s `scripts/guest/mkiso.sh`; its `tests/run.sh go` checks Go on that CD.
