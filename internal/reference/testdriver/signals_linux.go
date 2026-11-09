//go:build reference && linux

package testdriver

import (
	"os"
	"syscall"
)

func pause(p *os.Process) error  { return p.Signal(syscall.SIGSTOP) }
func resume(p *os.Process) error { return p.Signal(syscall.SIGCONT) }
