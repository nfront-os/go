// reading on in a directory that was emptied and removed meanwhile ends without an
// error or the file server dying (ext4srv read an entry past the directory's end)
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dir, err := os.MkdirTemp("", "dirremove")
	if err != nil {
		fmt.Println("FAIL dirremove", err)
		return
	}
	for i := range 500 {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file-with-a-long-name-%03d", i)), nil, 0644); err != nil {
			fmt.Println("FAIL dirremove", err)
			return
		}
	}
	f, err := os.Open(dir)
	if err != nil {
		fmt.Println("FAIL dirremove", err)
		return
	}
	first, err := f.ReadDir(1) // one read of entries: the server keeps its place
	if err != nil || len(first) != 1 {
		fmt.Println("FAIL dirremove: first read", len(first), err)
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		fmt.Println("FAIL dirremove: RemoveAll", err)
		return
	}
	rest, err := f.ReadDir(-1)
	f.Close()
	// the server must still answer
	if _, err2 := os.Stat(os.TempDir()); err2 != nil {
		fmt.Println("FAIL dirremove: server gone:", err2)
		return
	}
	// what comes after the removal may be what was read before it (os keeps a
	// buffer), but none of it may exist any more
	for _, e := range rest {
		if _, err := os.Lstat(filepath.Join(dir, e.Name())); err == nil {
			fmt.Println("FAIL dirremove: still there:", e.Name())
			return
		}
	}
	fmt.Println("ok dirremove: rest", len(rest), err)
}
