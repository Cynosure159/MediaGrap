//go:build !linux && !darwin

package library

import (
	"errors"
	"os"
)

const supplementalRootSupported = false

func openSupplementalDirectory(root *os.Root) (*os.File, error) {
	return nil, errors.New("supplemental filesystem confinement is unsupported on this platform")
}
