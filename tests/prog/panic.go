// an uncaught panic: the runtime prints a stack trace and exits with a status
package main

func f(n int) {
	if n == 0 {
		panic("gotest panic")
	}
	f(n - 1)
}

func main() { f(3) }
