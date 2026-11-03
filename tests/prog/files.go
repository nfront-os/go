// os: files, directories, env, exec, pipes
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func fail(f string, a ...any) { fmt.Printf("FAIL files: "+f+"\n", a...); os.Exit(1) }

func main() {
	dir, err := os.MkdirTemp("", "gotest")
	if err != nil {
		fail("%v", err)
	}
	defer os.RemoveAll(dir)
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("hello\n"), 0644); err != nil {
		fail("%v", err)
	}
	if b, err := os.ReadFile(p); err != nil || string(b) != "hello\n" {
		fail("read %q %v", b, err)
	}
	if err := os.Rename(p, filepath.Join(dir, "b.txt")); err != nil {
		fail("rename %v", err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil || len(ents) != 1 || ents[0].Name() != "b.txt" {
		fail("readdir %v %v", ents, err)
	}
	os.Setenv("GOTESTVAR", "x y")
	out, err := exec.Command("rc", "-c", "echo $GOTESTVAR; echo err >[1=2]").Output()
	if err != nil || string(out) != "x y\n" {
		fail("exec %q %v", out, err)
	}
	cmd := exec.Command("wc", "-l")
	cmd.Stdin = strings.NewReader("1\n2\n3\n")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil || strings.TrimSpace(b.String()) != "3" {
		fail("pipe %q %v", b.String(), err)
	}
	if err := exec.Command("rc", "-c", "exit oops").Run(); err == nil {
		fail("exit status lost")
	}
	fmt.Println("ok files")
}
