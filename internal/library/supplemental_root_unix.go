//go:build linux || darwin

package library

import (
	"os"

	"golang.org/x/sys/unix"
)

const supplementalRootSupported = true

func openSupplementalDirectory(root *os.Root) (*os.File, error) {
	return root.OpenFile(".", os.O_RDONLY|unix.O_DIRECTORY|unix.O_NONBLOCK|unix.O_NOFOLLOW, 0)
}
