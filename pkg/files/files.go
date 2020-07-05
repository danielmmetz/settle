package files

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielmmetz/settle/pkg/log"
	"github.com/danielmmetz/settle/pkg/store"
)

type Files []FileMapping

func (f Files) Ensure(ctx context.Context, logger log.Log, store store.Store) error {
	for _, mapping := range f {
		if err := mapping.ensure(ctx, logger, store); err != nil {
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

func (m FileMapping) ensure(ctx context.Context, logger log.Log, store store.Store) error {
	_, err := store.Lstat(m.Dst)
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
	return store.Symlink(ctx, m.Src, m.Dst)
}
