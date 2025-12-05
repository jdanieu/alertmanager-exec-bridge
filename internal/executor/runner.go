package executor

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

type Result struct {
	Command  string
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	TimedOut bool
}

// Run ejecuta el comando con timeout y devuelve stdout/stderr/exit code.
func Run(cmdPath string, cmdArgs []string, timeout time.Duration) (Result, error) {
	start := time.Now()

	cmd := exec.Command(cmdPath, cmdArgs...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	res := Result{
		Command: cmdPath,
		Args:    append([]string(nil), cmdArgs...), // copia defensiva
	}

	// Usamos un canal + goroutine para implementar el timeout sin
	// depender de context ligado a la request (queremos que sobreviva al handler).
	done := make(chan error, 1)

	go func() {
		done <- cmd.Run()
	}()

	var err error

	select {
	case err = <-done:
		// terminó antes del timeout
	case <-time.After(timeout):
		// timeout: intentamos matar el proceso
		_ = cmd.Process.Kill()
		res.TimedOut = true
		err = fmt.Errorf("command timed out after %s", timeout)
	}

	res.Duration = time.Since(start)
	res.Stdout = stdoutBuf.String()
	res.Stderr = stderrBuf.String()

	// Determinar exit code si es posible
	if exitErr, ok := err.(*exec.ExitError); ok {
		res.ExitCode = exitErr.ExitCode()
	} else if err == nil {
		res.ExitCode = 0
	} else {
		// error que no es ExitError (p.ej. no existe el binario)
		res.ExitCode = -1
	}

	return res, err
}
