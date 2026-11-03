// net: TCP over loopback, many clients at once, /net/cs lookups
package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("FAIL net listen", err)
		return
	}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				s, _ := bufio.NewReader(c).ReadString('\n')
				fmt.Fprint(c, "echo "+s)
			}(c)
		}
	}()
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, err := net.Dial("tcp", l.Addr().String())
			if err != nil {
				errs <- err
				return
			}
			defer c.Close()
			fmt.Fprintf(c, "%d\n", i)
			s, err := bufio.NewReader(c).ReadString('\n')
			if err != nil || s != fmt.Sprintf("echo %d\n", i) {
				errs <- fmt.Errorf("client %d: %q %v", i, s, err)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		fmt.Println("FAIL net", err)
		return
	}
	if addrs, err := net.LookupHost("localhost"); err != nil || len(addrs) == 0 {
		fmt.Println("FAIL net lookup", addrs, err)
		return
	}
	fmt.Println("ok net")
}
