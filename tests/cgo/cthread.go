// threads made by C (pthread_create) call into Go (the runtime's extra M for
// threads it did not make), several at once
package main

/*
#include <pthread.h>
#include <stdlib.h>
extern void goWork(int);
static void *thr(void *a) { goWork((int)(long)a); return 0; }
static int runThreads(int n) {
	pthread_t t[16];
	int i;
	for (i = 0; i < n; i++)
		if (pthread_create(&t[i], 0, thr, (void *)(long)i) != 0)
			return -1;
	for (i = 0; i < n; i++)
		pthread_join(t[i], 0);
	return 0;
}
*/
import "C"

import (
	"fmt"
	"sync/atomic"
)

var sum atomic.Int64

//export goWork
func goWork(i C.int) {
	s := 0
	for k := 0; k <= int(i)*1000; k++ {
		s += k
	}
	m := make([]byte, 1<<16) // allocation on a C thread
	m[0] = 1
	sum.Add(int64(s) + int64(m[0]))
}

func main() {
	if C.runThreads(8) != 0 {
		fmt.Println("FAIL cthread: pthread_create")
		return
	}
	var want int64
	for i := 0; i < 8; i++ {
		n := int64(i) * 1000
		want += n*(n+1)/2 + 1
	}
	if sum.Load() != want {
		fmt.Println("FAIL cthread", sum.Load(), want)
		return
	}
	fmt.Println("ok cthread")
}
