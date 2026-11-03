// a thread of C's own (it never calls Go) survives a note posted to it, and the
// program ends when main returns although that thread still runs (the runtime's
// note handler found no g there and ended the thread; exit left it running)
package main

/*
#include <pthread.h>
#include <unistd.h>
static volatile int ticks, cpid;
static void *worker(void *a) {
	cpid = getpid();
	for (;;) {
		ticks++;
		usleep(10000);
	}
	return 0;
}
static void startworker(void) {
	pthread_t t;
	pthread_create(&t, 0, worker, 0);
	pthread_detach(t);
}
static int getticks(void) { return ticks; }
static int getcpid(void) { return cpid; }
*/
import "C"

import (
	"fmt"
	"os"
	"time"
)

func main() {
	C.startworker()
	for C.getcpid() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/note", int(C.getcpid())), os.O_WRONLY, 0)
	if err != nil {
		fmt.Println("FAIL cnote:", err)
		return
	}
	f.Write([]byte("interrupt"))
	f.Close()
	time.Sleep(200 * time.Millisecond)
	t0 := C.getticks()
	time.Sleep(300 * time.Millisecond)
	if C.getticks() == t0 {
		fmt.Println("FAIL cnote: the C thread died of the note")
		return
	}
	// the worker still runs: main returns, the program must end (tests/cgo/run.sh
	// checks that no proc is left)
	fmt.Println("ok cnote")
}
