// C calls Go: an exported Go function called from C on a Go thread, with a Go
// string and slice through C
package main

/*
#include <stdint.h>
extern int goAdd(int, int);
extern void goNote(char *);
static int callGo(int n) {
	int i, s = 0;
	for (i = 0; i < n; i++)
		s = goAdd(s, i);
	goNote("from C");
	return s;
}
*/
import "C"

import "fmt"

var noted string

//export goAdd
func goAdd(a, b C.int) C.int { return a + b }

//export goNote
func goNote(s *C.char) { noted = C.GoString(s) }

func main() {
	s := C.callGo(1000)
	if s != 499500 || noted != "from C" {
		fmt.Println("FAIL callback", s, noted)
		return
	}
	fmt.Println("ok callback")
}
