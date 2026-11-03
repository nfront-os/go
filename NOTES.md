# 9front-go: Notizen zum Port

## Teil 1 (2026-11-03): Go 1.27.3 für 9front, amd64/386 gebaut, plan9/arm64 portiert

**Auftrag:** Go als Programmiersprache für alle Arches des gcc-Branchs (386, amd64, arm64);
eigenes Repo (wie 9front-rust), plan9/arm64 portieren. ELF statt a.out bringt keine
Geschwindigkeit (gleicher Code, nur anderes Ladeformat); Go-Programme bleiben a.out und
laufen damit auch auf 9front selbst.

**Aufbau:** `go/` = go1.27.3 (flacher Klon), Branch `plan9`. `scripts/build.sh`: make.bash
für den Host, dann `GOOS=plan9 GOARCH=$A go install std cmd`. `scripts/install.sh`: Baum
`/$objtype/go` mit src, lib, misc, api, `pkg/include` (Header des Assemblers: ohne sie
scheitert jedes `go build` im Gast an textflag.h), Befehle neu gelinkt mit `-trimpath` und
`-X runtime.defaultGOROOT=/$A/go` (sonst trüge `go` den Host-Pfad als GOROOT; über
`bind -b /$A/go/bin /bin` findet os.Executable den Baum nicht).

**amd64 (Gos eigener Port) auf dem gcc-Kernel:** läuft ohne Änderung. `tests/run.sh` 7/7.
`go run` im Gast: ~110 s beim ersten Mal (std übersetzen), danach `go build` 1,5 s.
Loopback: der Gast richtet nur ether0 ein; die Tests machen `ip/ipconfig loopback
/dev/null 127.1` wie 9front-rust.

**plan9/arm64 (neu):** nur die OS-Schicht, arm64-Compiler/Assembler/Linker gibt es.
Vorlage plan9/arm (g auch dort im Register).
- Kernel-Schnittstelle (9front `sys/src/9/arm64`): Systemaufruf-Nummer in R0, `SVC`; der
  Kernel liest die Argumente ab `sp+8` in 8-Byte-Fächern (`va_arg` von int liest die unteren
  4 Byte). Das ist bei einer Go-Funktion ohne Rahmen genau `arg+0(FP)`; nur gepackte
  int32-Paare (`dupfd(old, new int32)`) und seek (Zeiger auf das Ergebnis zuerst) brauchen
  einen eigenen Rahmen. Ergebnis in R0.
- Start: R0 = Tos, argc bei 0(RSP), argv danach; rt0 richtet RSP auf 16 aus.
- Notes (`trap.c notify`): R0 = Ureg*, Ureg* bei 8(RSP), Notentext bei 16(RSP), LR=0;
  zurück nur über noted. sigtramp wechselt auf den gsignal-Stack, ruft sighandler, dann
  noted. sigpanictramp legt das LR der fehlerhaften Funktion (sighandler hat es in r0
  gelegt) auf den Stack, wie preparePanic in signal_arm64.go, und springt in
  `sigpanic<ABIInternal>`. FP-Register überstehen Notes (`fpunotify`).
- Tos.pid bei Offset 64 (wie amd64). g nur in R28, kein TLS (`tls_arm64.h`: plan9 ohne
  cgo, load_g/save_g kehren vorher zurück).
- Linker: R_MAGIC (`4*28*28+7` | HDR_MAGIC = 0x8c47), Kopf 40 Byte, Text ab 0x10028, Daten
  auf der nächsten 64K-Grenze (der Kernel richtet R_MAGIC-Daten auf 64K aus,
  `sysproc.c`). Der alte Hplan9-Zweig in `cmd/link/internal/arm64/obj.go` (32-Byte-Kopf,
  4K) passte zu keinem Kernel.
- `internal/cpu`: nichts zu tun, `cpu_arm64_other.go` liest keine Systemregister (die
  9front aus EL0 nicht emuliert).
- `syscall` und das vendorte `golang.org/x/sys/plan9` (cmd): asm + zsyscall für arm64.
  Hinweis: `go mod vendor` in cmd würde die x/sys-Dateien entfernen; upstream bräuchte x/sys
  dieselben zwei Dateien.
- Ergebnis im arm64-Gast (ein Prozessor, `arm64/9qemu.u` vom 31.10.): `tests/run.sh` 7/7 beim
  ersten Lauf (auch nil-Zugriff → panic → recover, Notes über os/signal, TCP, exec).

**cgo:** geplant (Nutzerwunsch), siehe Plan: externes Linken mit 9front-gccs gcc (ELF wie die
übrigen gcc-Port-Programme), Start über APEs `_main` → Gos `main`, runtime/cgo über
pthreads von APE.
