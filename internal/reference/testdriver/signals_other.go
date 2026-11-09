//go:build reference && !linux

package testdriver

import (
	"fmt"
	"os"
)

func pause(*os.Process) error  { return fmt.Errorf("real process pause requires Linux/WSL") }
func resume(*os.Process) error { return fmt.Errorf("real process resume requires Linux/WSL") }
