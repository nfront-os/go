// writing to a loopback TCP connection whose other end is closed fails (9front's
// spliced loopback dropped the data silently: the writer never saw an error)
package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("FAIL tcpclosed", err)
		return
	}
	defer l.Close()
	ch := make(chan net.Conn, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			ch <- nil
			return
		}
		ch <- c
	}()
	c1, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println("FAIL tcpclosed", err)
		return
	}
	defer c1.Close()
	c2 := <-ch
	if c2 == nil {
		fmt.Println("FAIL tcpclosed: no accept")
		return
	}
	c2.Close()
	time.Sleep(100 * time.Millisecond)
	buf := make([]byte, 1<<20)
	for i := 0; i < 256; i++ {
		if _, err := c1.Write(buf); err != nil {
			fmt.Println("ok tcpclosed: write", i, "MB:", err)
			return
		}
	}
	fmt.Println("FAIL tcpclosed: 256 MB written to a closed connection")
}
