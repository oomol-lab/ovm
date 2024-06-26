// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package logger

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

var cs = make([]*Context, 0, 10)

func NewWithoutManage(p, n string) (*Context, error) {
	c := &Context{
		path: p,
		name: n,
		syncWriter: syncWriter{
			m:    sync.Mutex{},
			file: nil,
		},
	}
	if err := c.init(); err != nil {
		return nil, err
	}

	return c, nil
}

func New(p, n string) (*Context, error) {
	c := &Context{
		path: p,
		name: n,
		syncWriter: syncWriter{
			m:    sync.Mutex{},
			file: nil,
		},
	}
	if err := c.init(); err != nil {
		return nil, err
	}

	cs = append(cs, c)

	return c, nil
}

func NewWithoutStream(p, n string) (string, error) {
	c := &Context{
		path: p,
		name: n,
		syncWriter: syncWriter{
			m:    sync.Mutex{},
			file: nil,
		},
	}
	if err := c.init(); err != nil {
		return "", err
	}

	if err := c.file.Close(); err != nil {
		return "", fmt.Errorf("cannot close log file: %v", err)
	}

	return path.Join(p, n+".log"), nil
}

func CloseAll() {
	for _, c := range cs {
		_ = c.file.Close()
	}
}

type syncWriter struct {
	m    sync.Mutex
	file *os.File
}

func (w *syncWriter) write(b []byte) (n int, err error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.file.Write(b)
}

type Context struct {
	path string
	name string
	syncWriter
}

func (c *Context) init() error {
	max := 5
	for i := max - 1; i > 0; i-- {
		logName := c.name
		if i > 1 {
			logName += "." + strconv.Itoa(i)
		}
		logPath := path.Join(c.path, logName+".log")

		if _, err := os.Stat(logPath); err == nil {
			err := os.Rename(logPath, path.Join(c.path, c.name+"."+strconv.Itoa(i+1)+".log"))
			if err != nil {
				return fmt.Errorf("cannot rename log file: %v", err)
			}
		}

		if i == 1 {
			f, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("cannot open log file: %v", err)
			}
			c.file = f
		}
	}

	return nil
}

func (c *Context) base(t, message string) {
	d := time.Now().Format("2006-01-02 15:04:05.000")
	_, _ = c.write([]byte(fmt.Sprintf("%s [%s]: %s\n", d, t, message)))
}

func (c *Context) Info(message string) {
	c.base("INFO", message)
}

func (c *Context) Infof(format string, args ...any) {
	c.Info(fmt.Sprintf(format, args...))
}

func (c *Context) Warn(message string) {
	c.base("WARN", message)
	_ = c.file.Sync()
}

func (c *Context) Warnf(format string, args ...any) {
	c.Warn(fmt.Sprintf(format, args...))
}

func (c *Context) Error(message string) error {
	c.base("ERROR", message)
	_ = c.file.Sync()
	return fmt.Errorf(message)
}

func (c *Context) Errorf(format string, args ...any) error {
	return c.Error(fmt.Sprintf(format, args...))
}

func (c *Context) Close() {
	_ = c.file.Close()
}
