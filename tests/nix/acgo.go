// on a NIX kernel (../9front-nix): one goroutine moves its proc to an application core
// with rfork(RFCORE|RFCPREPAGE) and back with rfork(RFCCORE), without execac and without
// a change to Go. RFCPREPAGE: no prepaging (fork(2)); Go reserves its heap arenas with
// brk (a bss of about 1.2G, a few M used), prepaged it would take the memory. Run on a
// time-sharing core; the kernel needs a free AC. Checks:
//   - after LockOSThread and rfork(RFCORE|RFCPREPAGE) the goroutine's proc is on an AC
//     (/proc/n/core), the program's other procs stay on TCs; the user memory in use
//     grows by less than 64M (not prepaged, /dev/swap)
//   - it computes there (calls in the loop: Go on Plan 9 preempts only at calls) while
//     main works on a TC and sees it on the AC from outside, a GC included
//   - few round trips to the AC (/proc/n/status's NIX counters) for the computing
//   - a second rfork(RFCORE) with the only AC taken fails with an error
//   - rfork(RFCCORE) brings it back to a TC
//
// Prints "acgo ..." lines, then "ok acgo".
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	RFCPREPAGE = 1 << 16 // NIX: no prepaging
	RFCORE     = 1 << 17 // NIX: to a free application core
	RFCCORE    = 1 << 18 // NIX: back to a time-sharing core
)

func fail(f string, a ...any) {
	fmt.Printf("FAIL acgo: "+f+"\n", a...)
	os.Exit(1)
}

func procfile(pid int, f string) []string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/%s", pid, f))
	if err != nil {
		fail("%v", err)
	}
	return strings.Fields(string(b))
}

func core(pid int) string { return procfile(pid, "core")[0] }

// round trips to the AC: the first of the three NIX counters at the end of status
func trips(pid int) string { s := procfile(pid, "status"); return s[len(s)-3] }

// the application cores, from /dev/sysstat (the line ends with AC)
func acs() map[string]bool {
	b, err := os.ReadFile("/dev/sysstat")
	if err != nil {
		fail("%v", err)
	}
	m := map[string]bool{}
	for _, l := range strings.Split(string(b), "\n") {
		f := strings.Fields(l)
		if len(f) > 0 && f[len(f)-1] == "AC" {
			m[f[0]] = true
		}
	}
	return m
}

// the user pages in use, from /dev/swap ("n/m user")
func userpages() int {
	b, err := os.ReadFile("/dev/swap")
	if err != nil {
		fail("%v", err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[1] == "user" {
			var n int
			fmt.Sscanf(f[0], "%d/", &n)
			return n
		}
	}
	fail("no user line in /dev/swap")
	return 0
}

func rfork(flag uintptr) error {
	r, _, err := syscall.Syscall(syscall.SYS_RFORK, flag, 0, 0)
	if int(r) == -1 {
		return err
	}
	return nil
}

//go:noinline
func step(x uint64) uint64 { return x*6364136223846793005 + 1442695040888963407 }

func main() {
	ac := acs()
	if len(ac) == 0 {
		fail("no application core in /dev/sysstat")
	}
	if ac[core(os.Getpid())] {
		fail("main already on an AC: run it on a TC, not through execac")
	}
	type msg struct {
		pid int
		err error
	}
	there := make(chan msg)
	compute := make(chan bool)
	done := make(chan uint64)
	back := make(chan string)
	go func() {
		runtime.LockOSThread() // this goroutine keeps the proc, on the AC too
		pid := os.Getpid()
		if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/segment", pid)); err == nil {
			for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
				fmt.Println("acgo segment", strings.Join(strings.Fields(l), " "))
			}
		}
		err := rfork(RFCORE | RFCPREPAGE)
		there <- msg{pid, err}
		if err != nil {
			return
		}
		<-compute
		x := uint64(1)
		for t := time.Now(); time.Since(t) < 2*time.Second; {
			for range 1000000 {
				x = step(x)
			}
		}
		done <- x
		if err := rfork(RFCCORE); err != nil {
			fail("rfork(RFCCORE): %v", err)
		}
		os.Getpid() // a system call: the kernel takes the proc back on it
		back <- core(pid)
	}()
	u0 := userpages()
	m := <-there
	if m.err != nil {
		fail("rfork(RFCORE): %v", m.err)
	}
	c := core(m.pid)
	fmt.Println("acgo goroutine on core", c)
	if !ac[c] {
		fail("goroutine's proc on core %s, not an AC", c)
	}
	if mc := core(os.Getpid()); ac[mc] {
		fail("main's proc on the AC %s too", mc)
	}
	u1 := userpages()
	fmt.Println("acgo user pages", u0, "->", u1)
	if u1-u0 > 64<<20/4096 {
		fail("the bss was prepaged: %d pages more", u1-u0)
	}

	// the only AC is taken: another proc gets an error
	if len(ac) == 1 {
		e := make(chan error)
		go func() {
			runtime.LockOSThread()
			e <- rfork(RFCORE)
		}()
		if err := <-e; err == nil {
			fail("second rfork(RFCORE) with the AC taken succeeded")
		} else {
			fmt.Println("acgo second rfork(RFCORE):", err)
		}
	}

	t0 := trips(m.pid)
	close(compute)
	// main on a TC meanwhile: sees the goroutine on the AC, makes garbage, collects it
	var seen string
	var junk [][]byte
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		seen = core(m.pid)
		junk = append(junk, make([]byte, 1<<20))
	}
	runtime.GC()
	if !ac[seen] {
		fail("during the computing the goroutine's proc was on core %s", seen)
	}
	x := <-done
	t1 := trips(m.pid)
	fmt.Println("acgo computed", x != 0, "on core", seen, "round trips", t0, "->", t1, "gc ok", len(junk))
	if c := <-back; ac[c] {
		fail("after rfork(RFCCORE) still on the AC %s", c)
	} else {
		fmt.Println("acgo back on core", c)
	}
	fmt.Println("ok acgo")
}
