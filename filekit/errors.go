package filekit

import (
	"errors"
	"fmt"
)

// Common filesystem errors
var (
	ErrNotExist      = errors.New("file does not exist")
	ErrExist         = errors.New("file already exists")
	ErrPermission    = errors.New("permission denied")
	ErrClosed        = errors.New("file already closed")
	ErrNotDir        = errors.New("not a directory")
	ErrIsDir         = errors.New("is a directory")
	ErrNotEmpty      = errors.New("directory not empty")
	ErrInvalidName   = errors.New("invalid name")
	ErrInvalidOffset = errors.New("invalid offset")
	ErrInvalidWhence = errors.New("invalid whence")
	ErrNotSupported  = errors.New("operation not supported")
	ErrNotAllowed    = errors.New("operation not allowed")
	ErrInvalidSize   = errors.New("invalid file size")
	ErrNoSpace       = errors.New("no space left on device")
)

// PathError records an error and the operation and file path that caused it
type PathError struct {
	Op   string
	Path string
	Err  error
}

// Error implements the error interface
func (e *PathError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

// Unwrap returns the underlying error
func (e *PathError) Unwrap() error {
	return e.Err
}

// IsNotExist reports whether an error indicates that a file or directory
// does not exist
func IsNotExist(err error) bool {
	return errors.Is(err, ErrNotExist)
}

// IsExist reports whether an error indicates that a file or directory
// already exists
func IsExist(err error) bool {
	return errors.Is(err, ErrExist)
}

// IsPermission reports whether an error indicates that permission is denied
func IsPermission(err error) bool {
	return errors.Is(err, ErrPermission)
}
