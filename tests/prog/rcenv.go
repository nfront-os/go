// rcenv: the environment and exit status between Go and rc. tests/run.sh starts it
// from an rc that defines a function and a list and deletes another function
// (its /env file stays, empty): a child rc must get them back (an rc function in
// /env with a NUL after it was a syntax error, an empty one too), through
// os/exec and through syscall.Exec; a wait message comes without the kernel's
// quotes, as libc's wait gives it.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

const script = "rcenvfn; echo $#rcenvlist $rcenvlist(2)"
const want = "fn ok\n3 b\n"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "exec" {
		err := syscall.Exec("/bin/rc", []string{"rc", "-c", script}, os.Environ())
		fmt.Println("exec:", err)
		os.Exit(1)
	}
	ok := true
	fail := func(f string, a ...any) {
		fmt.Printf("FAIL rcenv: "+f+"\n", a...)
		ok = false
	}
	if _, found := os.LookupEnv("fn#rcenvgone"); !found {
		fail("no empty fn#rcenvgone in the environment")
	}
	if os.Getenv("fn#rcenvfn") == "" || os.Getenv("rcenvlist") != "a\x00b\x00c" {
		fail("start rcenv from rc with fn rcenvfn and rcenvlist=(a b c): %q %q",
			os.Getenv("fn#rcenvfn"), os.Getenv("rcenvlist"))
	}

	out, err := exec.Command("/bin/rc", "-c", script).CombinedOutput()
	if err != nil || string(out) != want {
		fail("os/exec: %q, %v", out, err)
	}
	// syscall.Exec replaces one proc of the program; the runtime's others stay
	// (a limit of its own), holding a pipe open: the output goes to a file
	if f, err := os.CreateTemp("", "rcenv"); err != nil {
		fail("%v", err)
	} else {
		cmd := exec.Command(os.Args[0], "exec")
		cmd.Stdout, cmd.Stderr = f, f
		err = cmd.Run()
		out, _ = os.ReadFile(f.Name())
		f.Close()
		os.Remove(f.Name())
		if err != nil || string(out) != want {
			fail("syscall.Exec: %q, %v", out, err)
		}
	}
	// a variable set in Go: a one-word list, and an empty one
	cmd := exec.Command("/bin/rc", "-c", "echo $#one $one $#empty")
	cmd.Env = append(os.Environ(), "one=x y", "empty=")
	if out, err = cmd.CombinedOutput(); err != nil || string(out) != "1 x y 1\n" {
		fail("Env: %q, %v", out, err)
	}

	cmd = exec.Command("/bin/rc", "-c", "exit 'it''s bad'")
	err = cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		fail("exit: %v", err)
	} else {
		w := ee.Sys().(*syscall.Waitmsg)
		m := fmt.Sprintf("rc %d: it's bad", cmd.Process.Pid)
		if w.Msg != m || ee.Error() != "exit status: "+m || ee.ExitCode() != 1 {
			fail("wait message %q, error %q, want %q", w.Msg, ee.Error(), m)
		}
	}
	cmd = exec.Command("/bin/rc", "-c", "status=''")
	if err = cmd.Run(); err != nil || cmd.ProcessState.Sys().(*syscall.Waitmsg).Msg != "" ||
		!cmd.ProcessState.Success() {
		fail("success: %v", err)
	}
	if strings.Contains(string(out), "syntax error") {
		fail("syntax error: %q", out)
	}
	if ok {
		fmt.Println("ok rcenv")
	}
}
