// goroutines, channels, sync, GC across all processors
package main

import (
	"fmt"
	"runtime"
	"sync"
)

func main() {
	n := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	var mu sync.Mutex
	sum := 0
	ch := make(chan int, 100)
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := make([]byte, 64<<10) // garbage for the GC
			b[0] = byte(i)
			ch <- i
			mu.Lock()
			sum += int(b[0]) - int(byte(i)) + i
			mu.Unlock()
		}(i)
	}
	got := 0
	for i := 0; i < 1000; i++ {
		got += <-ch
	}
	wg.Wait()
	runtime.GC()
	if sum != 499500 || got != 499500 {
		fmt.Println("FAIL goroutines", sum, got)
		return
	}
	fmt.Println("ok goroutines", n, "procs")
}
