// on a NIX kernel (../9front-nix): the core each of the program's procs runs on, from
// /proc/n/core (the first field). Run through execac, the main proc is on the
// application core; the procs Go makes with rfork run on time-sharing cores, as NIX
// starts a child there. Prints "core main N" and "core other N" per proc, then "ok core".
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
)

func core() string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/core", os.Getpid()))
	if err != nil {
		fmt.Println("FAIL core:", err)
		os.Exit(1)
	}
	return strings.Fields(string(b))[0]
}

func main() {
	runtime.LockOSThread()	// main stays on the first proc
	fmt.Println("core main", core())
	// procs of their own: each goroutine locked to a new M while the others wait
	var wg sync.WaitGroup
	start := make(chan bool)
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runtime.LockOSThread()
			<-start
			fmt.Println("core other", core())
		}()
	}
	close(start)
	wg.Wait()
	fmt.Println("ok core")
}
