// a note (interrupt) while a goroutine is in C reaches os/signal; a nil
// dereference in Go is still a recoverable panic in a cgo program
package main

/*
#include <unistd.h>
static void sleepms(int ms) { usleep(ms * 1000); }
*/
import "C"

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	func() {
		defer func() { recover() }()
		var p *int
		*p = 1
		fmt.Println("FAIL cgonote: no panic")
		os.Exit(1)
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		time.Sleep(200 * time.Millisecond)
		f, err := os.OpenFile(fmt.Sprintf("/proc/%d/note", os.Getpid()), os.O_WRONLY, 0)
		if err == nil {
			f.Write([]byte("interrupt"))
			f.Close()
		}
	}()
	C.sleepms(1000)
	select {
	case s := <-c:
		if s != syscall.Note("interrupt") {
			fmt.Println("FAIL cgonote: got", s)
			return
		}
	case <-time.After(10 * time.Second):
		fmt.Println("FAIL cgonote: no interrupt")
		return
	}
	fmt.Println("ok cgonote")
}
