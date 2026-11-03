// notes: os/signal gets an interrupt posted to the process; a nil dereference is a
// panic that recover catches (the runtime's note handler)
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	func() {
		defer func() {
			if r := recover(); r == nil {
				fmt.Println("FAIL note: no panic")
				os.Exit(1)
			}
		}()
		var p *int
		*p = 1
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/note", os.Getpid()), os.O_WRONLY, 0)
	if err != nil {
		fmt.Println("FAIL note", err)
		return
	}
	f.Write([]byte("interrupt"))
	f.Close()
	select {
	case s := <-c:
		if s != syscall.Note("interrupt") {
			fmt.Println("FAIL note: got", s)
			return
		}
	case <-time.After(10 * time.Second):
		fmt.Println("FAIL note: no interrupt")
		return
	}
	fmt.Println("ok note")
}
