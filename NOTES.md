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

## Teil 2 (2026-11-03): Mehrkern, Gos std-Tests im Gast, drei Fehler in 9front

- **Mehrkern** (`tests/prog/parallel.go`, `WANTCPU`): GOMAXPROCS = Prozessoren, CPU-Arbeit auf
  allen, Atomics, umkämpfter Mutex, Kanal-Pingpong, GC nebenher. 4 Prozessoren: amd64, 386,
  arm64 bestanden, Speedup 2,5–2,7. arm64 startete bis dahin nur einen Prozessor: der Gast
  bootete einen alten `9qemu.u` ohne den SMP-Fix des Baums (585787b25); nach `ARCH=arm64` und
  `scripts/guest/build-boot.sh` dort alle vier. `tests/guest.sh` nimmt jetzt auch auf arm64 4.
- **install.sh**: auch die von cmd/dist erzeugten, in Git ignorierten Dateien (`zzipdata.go`
  der tzdata, `zversion.go`, `zbootstrap.go`, `zdefaultcc.go`); ohne zzipdata baute `time`s
  Test nicht („undefined: zipdata“).
- **toolchain.sh**: `::1` als Loopback (`ip/ipconfig loopback /dev/null ::1 /128`, ipconfig(8)),
  Prüfmuster mit Tab nach „ok“.
- **Fehler in 9front** (im gcc-Branch behoben, Patches 0018–0020 dort):
  - ext4srv starb bei `go test` (arm64): ein Lesen von 0 Byte ließ den count von Rread
    uninitialisiert (> msize, lib9p brach ab).
  - ext4srv: remove und rename ohne Schreibrecht im Elternverzeichnis
    (`TestRemoveAllButReadOnlyAndPathError`).
  - TCP-Loopback: Schreiben an eine geschlossene Gegenseite verschwand still
    (`TestWritevError`); jetzt „i/o on hungup channel“, `tests/prog/tcpclosed.go`.
- **Go-Tests angepasst** (go/, Branch plan9): `TestOpenError` nimmt 9fronts Form der Fehler
  (Kernel `namec`: Name hinter dem Fehler); `TestIPv6WriteMsgUDPAddrPortTargetAddrIPVersion`
  auf plan9 übersprungen (WriteMsgUDP gibt es nicht).
- **Ergebnis** (`go test -short` errors strings strconv sort fmt os time sync net, toolchain.sh): amd64, 386 und arm64 je 14/14, run.sh je 10/10 mit 4 Prozessoren.
  arm64 vor den Fixes: dieselben Fehler wie amd64, also kein Fehler des arm64-Ports.
- **Vierter Fehler in 9front** (gcc-Branch, Patch 0021): ext4srv starb, wenn ein offenes
  Verzeichnis nach dem Leeren/Löschen weitergelesen wurde (Gos os-Tests auf arm64:
  `fault read addr=0x6`). `tests/prog/dirremove.go` zeigt es auch auf amd64 (stürzte sofort
  ab, jetzt Ende des Verzeichnisses; die Einträge danach kommen aus Gos 64-KB-Puffer, keiner
  existiert noch). `tests/run.sh` legt /tmp dafür auf eine ext4-Platte.
- **Falle arm64**: nach jedem Build mit `ARCH=arm64` ist `9qemu` neuer als `9qemu.u`, und
  `boot-cli.sh` bootet absichtlich nicht (alter Kernel). Erst `scripts/guest/build-boot.sh`.
  run.sh und toolchain.sh sagen jetzt, wenn der Gast nicht hochkam (vorher nur „0/14“).

## Teil 3 (2026-11-03): cgo auf amd64

Nutzerwunsch: cgo mit einplanen; amd64 zuerst. Go hatte auf Plan 9 nie cgo.
- **Linken**: cgo verlangt auf plan9 externes Linken (`internal/platform.MustLinkExternal`);
  `cmd/link` schreibt dann ein ELF-Objekt (Elfinit nach `determineLinkMode`, asmbElf statt
  asmbPlan9), 9front-gccs gcc linkt mit `plan9.ld` und APE ein ELF. Start: APEs `_main` →
  `_callmain` → libgccs `__wrap_main` → `__plan9_mainstack` (main auf 16 MB Heap-Stack) →
  Gos `main` (asm_amd64.s) → `rt0_go`.
- **TLS**: Gos TLS-Zeiger hieß `_privates`; das ist in einem cgo-Programm APEs Feld und Slot 0
  gehört dem pthread. Jetzt `_goprivates` (asm6, rt0_plan9_*.s); `gcc_plan9.c` setzt ihn in
  x_cgo_inittls auf `&_privates[2]` — g liegt bei `-8(_goprivates)` (Tlsoffset), also in Slot 1.
  Erster Versuch `&_privates[1]` traf Slot 0 (das struct __pthread als g: Absturz in sigtramp).
  `_goprivates` und `_tos` sind für die C-Seite exportiert (`cgo_export_static`), sonst lokal
  im go.o (APEs `_tos` wäre ein anderes gewesen).
- **Heap**: Go und APEs malloc nahmen beide brk ab `end`. In einem cgo-Programm ist Gos Heap ein
  eigenes Segment (`segattach "memory"` bei `heapSegBase`, 16 TB auf amd64/arm64, 1 GB auf 386;
  Wachstum mit segbrk), angelegt in osinit, bevor es andere Procs gibt (ein RFMEM-Kind teilt nur
  die Segmente, die beim fork da waren, aber deren Wachstum).
- **Fehler im Baum** (gcc-Branch): `plan9.ld` setzte `end` vor Gos `.noptrbss` (verwaiste
  Sektion) — malloc überschrieb Gos Variablen; APEs pthreads erbten die `_privates` des
  Erzeugers — ein C-Thread hielt das g seines Erzeugers für seins. Beides behoben.
- **Runtime**: runtime/cgos POSIX-Teil (pthreads) baut für plan9; `cgoBindM` erlaubt (pthread-
  Schlüssel mit Destruktor gibt es in APE); `minit` setzt procid für Threads von runtime/cgo.
- **errno**: cgo liefert `syscall.Errno` (auf plan9 nur ein Platzhalter, „errno N“) mit APEs
  Zahlen (ENOENT 20).
- **Build-Cache**: mit einer Release-VERSION (go1.27.3) erkennt cmd/go den Compiler nur an der
  Version; nach der Umbenennung kamen Objekte des alten Assemblers aus dem Cache („relocation
  target _privates not defined“). build.sh/install.sh/Tests haben jetzt `build/gocache`, den
  `HOST=1` leert.
- **Nativ**: braucht 9front-gccs nativen gcc ohne die alten compat-Header: `out-gcc/$objtype`
  (30.10.); `out/amd64` (22.10.) hat noch `x86_64-pc-plan9/include/stdlib.h` & co., die mit dem
  heutigen APE kollidieren (lldiv_t doppelt).
- **Ergebnis amd64**: `tests/cgo/run.sh` 5/5 (Cross), `tests/cgo/native.sh` 5/5 (go und gcc im
  Gast); ohne cgo weiter run.sh 10/10 auf amd64 und 386.

## Teil 4 (2026-11-03): cgo auf 386

- Wie amd64, dazu nur: `main` für APEs Start in `rt0_plan9_386.s` (wie `rt0_linux_386.s`:
  Rücksprungadresse weg, `rt0_go`); g bei `-4(_goprivates)`, also auch mit `&_privates[2]` in
  Slot 1.
- **Speicheraufteilung** (Hinweis eines Nebenagenten: das Heap-Segment bei 1 GB ließ C nur ~1 GB):
  Nutzeradressraum bis knapp 0xe0000000 (16 MB Stack darunter), ein Segment höchstens 1984 MB
  (SEGMAPSIZE). `heapSegBase` jetzt 0x70000000: brk (APEs malloc) und Go je ~1,7 GB.
  `tests/cgo/cgomem.go`: C 1,5 GB per malloc, danach Go 1 GB — besteht (mit 1 GB als Grenze
  hätte C bei ~1 GB aufgehört).
- **Ergebnis**: `tests/cgo/run.sh` 6/6 auf 386 und amd64, `native.sh` 6/6 auf 386 (go und gcc im
  Gast, 9front-gccs out-gcc/386); ohne cgo run.sh 386 10/10.
