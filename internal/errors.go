package internal

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnsupportedSource  = errors.New("unsupported source")
	ErrAlreadyInstalled   = errors.New("already installed")
	ErrNotInstalled       = errors.New("not installed")
	ErrCorruptMetadata    = errors.New("corrupt metadata")
	ErrDestinationExists  = errors.New("destination exists")
	ErrPathTraversal      = errors.New("path traversal detected")
	ErrIntegrityCheckFail = errors.New("integrity check failed")
)

func WrapInvalidInput(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}

func WrapUnsupportedSource(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrUnsupportedSource, fmt.Sprintf(format, args...))
}

func WrapAlreadyInstalled(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrAlreadyInstalled, fmt.Sprintf(format, args...))
}

func WrapNotInstalled(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrNotInstalled, fmt.Sprintf(format, args...))
}
