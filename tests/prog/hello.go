package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf("hello from Go %s on %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
