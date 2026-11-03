// time: monotonic clock, timers, tickers, sleep
package main

import (
	"fmt"
	"time"
)

func main() {
	t0 := time.Now()
	time.Sleep(200 * time.Millisecond)
	d := time.Since(t0)
	tk := time.NewTicker(20 * time.Millisecond)
	for i := 0; i < 5; i++ {
		<-tk.C
	}
	tk.Stop()
	select {
	case <-time.After(50 * time.Millisecond):
	case <-time.After(5 * time.Second):
		fmt.Println("FAIL timer: wrong timer fired")
		return
	}
	if d < 200*time.Millisecond || d > 2*time.Second || time.Now().Year() < 2025 {
		fmt.Println("FAIL timer", d, time.Now())
		return
	}
	fmt.Println("ok timer")
}
