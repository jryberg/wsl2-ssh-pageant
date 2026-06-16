package main

// Console-attachment logic ported from github.com/apenwarr/fixconsole
// (BSD-3-Clause). Inlined here to avoid depending on the unmaintained
// apenwarr/fixconsole and apenwarr/w32 modules; it relies only on the stdlib
// and the maintained golang.org/x/sys/windows.

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func attachConsole() error {
	const attachParentProcess = ^uintptr(0)
	proc := syscall.MustLoadDLL("kernel32.dll").MustFindProc("AttachConsole")
	r1, _, err := proc.Call(attachParentProcess)
	if r1 == 0 {
		errno, ok := err.(syscall.Errno)
		if ok && errno == windows.ERROR_INVALID_HANDLE {
			// console handle doesn't exist; not a real error, but the
			// console handle will be invalid.
			return nil
		}
		return err
	}
	return nil
}

var oldStdin, oldStdout, oldStderr *os.File

// FixConsoleIfNeeded reattaches and repairs the standard handles for a
// "-H windowsgui" binary so that stdin/stdout/stderr work for the agent
// protocol. See the original apenwarr/fixconsole for the full rationale on
// why each step is needed and the matrix of shells this fixes.
func FixConsoleIfNeeded() error {
	// Retain the original console objects, to prevent Go from automatically
	// closing their file descriptors when they get garbage collected.
	// You never want to close file descriptors 0, 1, and 2.
	oldStdin, oldStdout, oldStderr = os.Stdin, os.Stdout, os.Stderr

	stdin, _ := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	stdout, _ := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	stderr, _ := syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)

	var invalid syscall.Handle
	con := invalid

	if stdin == invalid || stdout == invalid || stderr == invalid {
		err := attachConsole()
		if err != nil {
			return fmt.Errorf("attachconsole: %v", err)
		}

		if stdin == invalid {
			stdin, _ = syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
		}
		if stdout == invalid {
			stdout, _ = syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
			con = stdout
		}
		if stderr == invalid {
			stderr, _ = syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)
			con = stderr
		}
	}

	if con != invalid {
		// Make sure the console is configured to convert
		// \n to \r\n, like Go programs expect.
		h := windows.Handle(con)
		var st uint32
		err := windows.GetConsoleMode(h, &st)
		if err != nil {
			return fmt.Errorf("GetConsoleMode: %v", err)
		}
		err = windows.SetConsoleMode(h, st&^windows.DISABLE_NEWLINE_AUTO_RETURN)
		if err != nil {
			return fmt.Errorf("SetConsoleMode: %v", err)
		}
	}

	if stdin != invalid {
		os.Stdin = os.NewFile(uintptr(stdin), "stdin")
	}
	if stdout != invalid {
		os.Stdout = os.NewFile(uintptr(stdout), "stdout")
	}
	if stderr != invalid {
		os.Stderr = os.NewFile(uintptr(stderr), "stderr")
	}
	return nil
}
