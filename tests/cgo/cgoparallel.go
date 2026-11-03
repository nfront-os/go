// many goroutines in C at once (the runtime makes threads through pthread_create),
// C's malloc and Go's GC side by side, errno back to Go
package main

/*
#include <errno.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
static long work(int n) {
	long s = 0;
	int i;
	for (i = 0; i < n; i++) {
		char *p = malloc(256 + i % 1000);
		memset(p, i, 256);
		s += p[0] & 0xff;
		free(p);
	}
	return s;
}
static int blockms(int ms) { usleep(ms * 1000); return ms; }
static int fail(void) { errno = ENOENT; return -1; }
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
)

func main() {
	var wg sync.WaitGroup
	res := make([]int64, 32)
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			C.blockms(50) // 32 goroutines blocked in C: threads of their own
			res[i] = int64(C.work(20000))
			_ = make([]byte, 1<<20)
		}()
	}
	go func() {
		for range 20 {
			runtime.GC()
		}
	}()
	wg.Wait()
	for i := 1; i < 32; i++ {
		if res[i] != res[0] {
			fmt.Println("FAIL cgoparallel: results differ", res[0], res[i])
			return
		}
	}
	_, err := C.fail()
	if err != syscall.Errno(C.ENOENT) { // C's errno, APE's number
		fmt.Println("FAIL cgoparallel: errno", err)
		return
	}
	fmt.Println("ok cgoparallel", runtime.GOMAXPROCS(0), "procs")
}
