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

## Teil 5 (2026-11-03): cgo auf arm64; Notes und Ende mit Threads von C

- **arm64**: g liegt in R28; C sichert R28 (callee-saved), darf es aber zwischendurch benutzen.
  `load_g`/`save_g` für plan9 (tls_arm64.s) halten g in einem cgo-Programm zusätzlich in APEs
  `_privates[1]` (`-8(_goprivates)`, wie amd64); `sigtramp` lädt es dort (eine Note mitten in C).
  `_goprivates` jetzt auch in rt0_plan9_arm64.s. `main` gibt es in asm_arm64.s schon.
- **Adressraum arm64**: 9fronts Nutzerraum ist nur 32 GB (EVASHIFT 36), ein Segment höchstens 8 GB
  (SEGMAPSIZE 8192): `heapSegBase` 8 GB (vorher 16 TB — außerhalb).
- **cmd/go** überschrieb `bin/plan9_arm64/go` nicht („not an object file“): es kannte die a.out-
  Magic von arm64 nicht (0x00008c47); ebenso `debug/plan9obj` (unexportiert, keine API-Änderung).
- **Notes auf Threads von C** (Hinweis eines Nebenagenten): ein reiner C-Thread hat kein g,
  `sigtramp` rief `badsignal2`, der Thread endete — bei jeder Note an die Note-Gruppe (Entf).
  Jetzt `badnote`: in einem cgo-Programm eine eigene Falle („sys: …“) mit dem Standard (der Proc
  stirbt wie in C), jede andere Note NCONT (Gos Threads bekommen Gruppen-Notes selbst).
- **Ende**: Gos `exit` schickt „go: exit“ nur an seine Ms; C-Threads liefen nach dem Ende von
  main weiter. Jetzt ruft exit in einem cgo-Programm APEs `__pthread_killothers` (neu, gcc-Branch;
  proc(3) ctl „kill“ für jeden pthread). Gegenprobe ohne: ein Proc blieb.
- **Rennen beim Ende** (arm64, cgoparallel: ein Proc in Semacquire blieb): (1) exit tötete erst die
  pthreads und setzte danach `exiting` — ein M dazwischen parkte; jetzt `goexitsall` zuerst.
  (2) APE: ein Thread steht mit pid 0 in der Liste, bis er läuft; `killothers` übersprang ihn.
  Jetzt setzt killothers erst `exiting`, ein neuer Thread speichert erst seine pid und liest dann
  `exiting` (SEQ_CST): einer sieht den anderen.
- `tests/cgo/cnote.go` (C-Thread übersteht eine Note, Programm endet), run.sh listet übrige Procs.
- **Ergebnis**: tests/cgo/run.sh 8/8 auf amd64, 386 und arm64 (arm64 dreimal hintereinander); native.sh 7/7 auf arm64 (go und gcc im Gast, 9front-gccs out-gcc/arm64).

## 2026-11-04: 9front-gccs out neu, CD mit cgo
- 9front-gccs `out/{amd64,386,arm64}` neu gebaut (gegen den gcc-Branch von heute); native.sh nimmt jetzt `out` statt `out-gcc`: amd64, 386 und arm64 je 7/7. Die alten compat-Header (`$target/include`, `libapecompat.a`, Reste von Installationen vor dem 22.10.) aus `out/{amd64,386}` entfernt.
- Install-CD amd64 mit diesem Go und gcc: tests/go im gcc-Branch 7/7, darin cgo mit dem gcc der CD. Dafür in APE behoben (gcc-Branch, Patch 0022): lib9ps „file not found“ war kein ENOENT; gcc in `/tmp` (ramfs) brach beim Suchen der Header ab.

## 2026-11-04: Go unter NIX (../9front-nix)
- `tests/nix.sh`: der nix-Kernel (Branch nix, auf den heutigen gcc rebased) mit `*nixac=1` von 4 CPUs; die Programme von tests/prog laufen normal (TC) und per `execac -c 3` auf dem AC, dazu `tests/nix/core.go` (Kern jedes Procs aus `/proc/n/core`). amd64 22/22, arm64 22/22 (TCG, 9qemu.u von build-boot.sh neu gemacht, sonst bootet der alte Kernel). Gos Haupt-Proc läuft auf dem AC (`core main 3`), die Procs, die Go per rfork macht, auf den TCs (`core other 0`): NIX startet ein Kind eines AC-Prozesses auf einem TC. Gos `NumCPU` zählt die Zeilen von `/dev/sysstat`, also auch den AC (4). Keine Änderung an Go nötig.
- Falle: guest.sh bricht nach 600 s ab und gibt dann nichts aus; jedes Programm zweimal dauert länger (`GUEST_TIMEOUT` 1800 in nix.sh).
- `tests/nix/acgo.go` (in tests/nix.sh, nur auf einem TC): eine Goroutine holt sich ohne execac und ohne Änderung an Go den AC — `runtime.LockOSThread`, dann `rfork(RFCORE|RFCPREPAGE)` per `syscall.Syscall(SYS_RFORK, …)`; sie rechnet dort (main sieht sie von einem TC aus auf dem AC, ein GC läuft), ein zweites RFCORE scheitert („no free application core“), `rfork(RFCCORE)` bringt sie zurück. ~2000 Hin- und Rückwege in 2 s Rechnen (Gos Zeitabfragen, Seiten). **RFCPREPAGE nötig**: Go reserviert die Heap-Arenen per brk (Bss bis 0x4c000000, ~1,2 GB), Prepage hätte alles angelegt. amd64 23/23, arm64 23/23 (dort ~420 Hin- und Rückwege).
- **Zwei NIX-Fehler damit gefunden, auf `nix` behoben**: (1) eine Note, die runacores Schlaf unterbrach, ging verloren, wenn der AC gerade mit einem Systemaufruf herauskam (Gos exit: `go: exit` an den AC-Proc auf dem Weg in semacquire → hing für immer; `execac … note` hing so beim 4. Durchgang). (2) `rfork(RFCORE)` legte die ganze dünne Bss an, der Kernel tötete den Prozess; jetzt RFCORE|RFCPREPAGE ohne Prepage, und Prepage hört bei knappem Speicher auf.

## 2026-11-05: rc-Funktionen in /env, Exit-Meldung ohne Quotes (Patch 0006)
Gefunden mit `../9front-mcp` (MCP-Server, der rc-Befehle startet); beide Stellen sind Gos eigener Code, upstream ebenso.
- **Umgebung**: `syscall/exec_plan9.go` schrieb jedem Wert eine NUL an. Für Variablen stimmt das (rc: jedes Wort mit NUL; goenvs nimmt die letzte weg), aber eine Funktion (`fn#name`) ist rc-Quelltext, den rc mit `. -bq` liest → in jedem rc, das Go startete, `/env/fn#sigexit:1: syntax error`. `envValue` schreibt beides wie rc, für fork/exec und `syscall.Exec` gleich (Exec schrieb gar keine NUL). Falle beim ersten Versuch: rc lässt nach `fn x` eine **leere** `/env/fn#x` stehen → 0 Bytes; ein nil-Zeiger beim pwrite gibt auch bei Länge 0 `sys: bad address in syscall` (jeder Start eines Kindes starb, tests/prog/files auch) → Puffer mit Kapazität ≥ 1.
- **Exit-Meldung**: `Waitmsg.Msg` behielt das `%q` des Kernels (`'rc 123: it''s'`, nur `''` wurde zu ""). Jetzt die Meldung selbst wie libcs wait (tokenize), `ExitError.Error()` = `exit status: rc 123: it's`. **Bewusst anders als upstream**: dessen `TestExitStatus` erwartete die Quotes (angepasst). Jedes Go-Programm, das mit diesem Go gebaut wird, sieht die Meldung ohne Quotes; Code, der selbst entquotet, bleibt richtig, solange er ungequotete Meldungen durchlässt. Bei einem neuen Go-Release muss der Patch mit. `cmd/vendor/golang.org/x/sys/plan9` hat dieselbe Stelle noch (vendored, nur von cmd benutzt).
- **Nicht behoben, schon vorher so**: nach `syscall.Exec` laufen die übrigen Procs des Go-Programms weiter (Semacquire, für immer) und halten geerbte fds offen — ein Elternteil, der aus einer Pipe liest, sieht kein EOF. Exec ersetzt nur den aufrufenden Proc; die anderen müssten vorher beendet werden.
- `tests/prog/rcenv.go` (run.sh startet es aus einem rc mit Funktion, Liste und gelöschter Funktion): Kind-rc über os/exec und syscall.Exec, Env aus Go, Exit-Meldung mit Quote darin, Erfolg = "". amd64 11/11.
