package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielmmetz/settle/pkg/log"
)

type Files []FileMapping

func (f Files) Ensure(logger log.Log) error {
	for _, mapping := range f {
		if err := mapping.ensure(logger); err != nil {
			return err
		}
	}
	return nil
}

func (m *FileMapping) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var intermediary struct {
		Src string
		Dst string
	}
	if err := unmarshal(&intermediary); err != nil {
		return err
	}

	absSrc, err := filepath.Abs(intermediary.Src)
	if err != nil {
		return fmt.Errorf("unable to resolve to absolute path: %w", err)
	}
	m.Src = absSrc
	m.Dst = intermediary.Dst
	return nil
}

type FileMapping struct {
	Src string
	Dst string
}

func (m FileMapping) ensure(logger log.Log) error {
	_, err := os.Lstat(m.Dst)
	if errors.Is(err, os.ErrNotExist) {
		// do nothing
	} else if err != nil {
		return err
	} else if err == nil {
		logger.Debug("file exists, deleting it: %s", m.Dst)
		if err := os.Remove(m.Dst); err != nil {
			return err
		}
	}
	logger.Debug("symlinking %v to %v", m.Src, m.Dst)
	return os.Symlink(m.Src, m.Dst)
}
