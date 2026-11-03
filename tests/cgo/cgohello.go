package main

/*
#include <stdio.h>
#include <stdlib.h>

static int add(int a, int b) { return a + b; }
static void say(const char *s) { printf("C says: %s\n", s); fflush(stdout); }
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func main() {
	s := C.CString("hello from cgo")
	C.say(s)
	C.free(unsafe.Pointer(s))
	if C.add(40, 2) != 42 {
		fmt.Println("FAIL cgohello")
		return
	}
	fmt.Println("ok cgohello")
}
