// C and Go both take a lot of address space (pages touched only at the start):
// C's malloc 1.5 GB in 64 MB pieces (the break), then Go 1 GB (its heap segment);
// on 386 the split at heapSegBase must leave both room
package main

/*
#include <stdlib.h>
static int cbig(int n, int mb) {
	int i;
	for (i = 0; i < n; i++) {
		char *p = malloc((size_t)mb << 20);
		if (p == NULL)
			return i;
		p[0] = 1;
	}
	return n;
}
*/
import "C"

import "fmt"

var keep [][]byte

func main() {
	if n := C.cbig(24, 64); n != 24 {
		fmt.Println("FAIL cgomem: C's malloc stopped after", int(n)*64, "MB")
		return
	}
	for i := 0; i < 8; i++ {
		b := make([]byte, 128<<20)
		b[0] = 1
		keep = append(keep, b)
	}
	fmt.Println("ok cgomem: C 1536 MB, Go 1024 MB")
}
