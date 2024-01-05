// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package pidlock

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

type Context struct {
	fh       *os.File
	filePath string
}

func New(p string) *Context {
	return &Context{
		filePath: p,
	}
}

func (l *Context) TryLock() error {
	fh, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create pid file failed: %w", err)
	}

	if err := syscall.Flock(int(fh.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("lock pid file failed: %w", err)
	}

	if _, err := fh.WriteString(fmt.Sprintf("%d", os.Getpid())); err != nil {
		return fmt.Errorf("write pid file failed: %w", err)
	}

	l.fh = fh

	return nil
}

func (l *Context) Unlock() {
	if l.fh == nil {
		return
	}

	_ = syscall.Flock(int(l.fh.Fd()), syscall.LOCK_UN)
	_ = l.fh.Close()
	_ = os.RemoveAll(l.filePath)
}

func (l *Context) Owner() (int, error) {
	file, err := os.ReadFile(l.filePath)
	if err != nil {
		return 0, err
	}

	p, err := strconv.Atoi(string(file))
	if err != nil {
		return 0, err
	}

	return p, nil
}
