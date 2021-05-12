package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Files []FileMapping

type FileMapping struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

func (f *Files) Ensure(ctx context.Context) error {
	if f == nil {
		return nil
	}

	for _, m := range *f {
		_, err := os.Lstat(m.Dst)
		if errors.Is(err, os.ErrNotExist) {
			// do nothing
		} else if err != nil {
			return err
		} else if err == nil {
			resolvedLink, err := os.Readlink(m.Dst)
			if err == nil && resolvedLink == m.Src {
				continue
			}
			fmt.Println("file exists, deleting it:", m.Dst)
			if err := os.Remove(m.Dst); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(filepath.Dir(m.Dst), 0755); err != nil {
			return fmt.Errorf("error making intermediate directories for %s: %w", m.Dst, err)
		}
		fmt.Printf("symlinking %s to %s\n", m.Src, m.Dst)
		if err := os.Symlink(m.Src, m.Dst); err != nil {
			return fmt.Errorf("error writing symlink from %s to %s: %w", m.Src, m.Dst, err)
		}
	}
	return nil
}

func (m *FileMapping) UnmarshalJSON(b []byte) error {
	var intermediary struct {
		Src string
		Dst string
	}
	if err := json.Unmarshal(b, &intermediary); err != nil {
		return err
	}
	absSrc, err := filepath.Abs(intermediary.Src)
	if err != nil {
		return fmt.Errorf("unable to resolve to absolute path: %w", err)
	}
	resolvedDst, err := expandTilde(intermediary.Dst)
	if err != nil {
		return fmt.Errorf("unable to resolve destination path: %w", err)
	}
	m.Src = absSrc
	m.Dst = resolvedDst
	return nil
}

func expandTilde(path string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home dir: %w", err)
	}

	components := strings.Split(path, string(os.PathSeparator))
	for i, component := range components {
		if component == "~" {
			components[i] = home
		}
	}
	return filepath.Join(components...), nil
}
