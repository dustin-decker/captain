package mocks

import (
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
)

// FileSystem is a mocked implementation of 'cli.FileSystem'.
type FileSystem struct {
	MockOpen func(name string) (fs.File, error)
}

// Open either calls the configured mock of itself or returns an error if that doesn't exist.
func (f *FileSystem) Open(name string) (fs.File, error) {
	if f.MockOpen != nil {
		return f.MockOpen(name)
	}

	return nil, errors.NewConfigurationError("MockOpen was not configured")
}