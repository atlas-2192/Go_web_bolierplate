//go:build unix

package cmd

import (
	"golang.org/x/sys/unix"
)

func init() {
	shutdownSignals = append(shutdownSignals, unix.SIGTERM, unix.SIGQUIT)
}
