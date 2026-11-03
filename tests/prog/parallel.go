// several processors: GOMAXPROCS from the machine's processors, CPU-bound goroutines
// on all of them (atomics, a contended mutex, channel ping-pong between procs, GC
// while they run); prints the speedup of the parallel work over one goroutine
package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func work(n int) uint64 {
	x := uint64(n)
	for i := 0; i < 20_000_000; i++ {
		x = x*6364136223846793005 + 1442695040888963407
	}
	return x
}

func fail(f string, a ...any) { fmt.Printf("FAIL parallel: "+f+"\n", a...); os.Exit(1) }

func main() {
	ncpu := runtime.NumCPU()
	procs := runtime.GOMAXPROCS(0)
	if procs != ncpu {
		fail("GOMAXPROCS %d, NumCPU %d", procs, ncpu)
	}
	if want := os.Getenv("WANTCPU"); want != "" && want != fmt.Sprint(ncpu) {
		fail("NumCPU %d, want %s", ncpu, want)
	}

	// one goroutine, then one per processor
	t0 := time.Now()
	r0 := work(1)
	one := time.Since(t0)
	t0 = time.Now()
	res := make([]uint64, procs)
	var wg sync.WaitGroup
	for i := range procs {
		wg.Add(1)
		go func() { defer wg.Done(); res[i] = work(1) }()
	}
	wg.Wait()
	all := time.Since(t0)
	for i, r := range res {
		if r != r0 {
			fail("result %d differs", i)
		}
	}

	// atomics and a contended mutex from all processors, GC running meanwhile
	var cnt atomic.Int64
	var mu sync.Mutex
	plain := 0
	stop := make(chan bool)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
				time.Sleep(time.Millisecond)
			}
		}
	}()
	for range 4 * procs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100_000 {
				cnt.Add(1)
				mu.Lock()
				plain++
				mu.Unlock()
				if plain%1000 == 0 {
					_ = make([]byte, 4096)
				}
			}
		}()
	}
	wg.Wait()
	close(stop)
	if n := int64(4 * procs * 100_000); cnt.Load() != n || int64(plain) != n {
		fail("counts %d %d, want %d", cnt.Load(), plain, n)
	}

	// ping-pong between goroutines (on different procs when there are several)
	ping, pong := make(chan int), make(chan int)
	go func() {
		for v := range ping {
			pong <- v + 1
		}
		close(pong)
	}()
	v := 0
	for range 100_000 {
		ping <- v
		v = <-pong
	}
	close(ping)
	if v != 100_000 {
		fail("ping-pong %d", v)
	}
	speedup := float64(one) * float64(procs) / float64(all)
	fmt.Printf("ok parallel %d procs, speedup %.1f (one %v, %d at once %v)\n", procs, speedup, one.Round(time.Millisecond), procs, all.Round(time.Millisecond))
	if procs > 1 && speedup < 1.3 {
		fmt.Println("FAIL parallel: no speedup with several processors")
	}
}
