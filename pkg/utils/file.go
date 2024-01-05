// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Copy(src, dst string) error {
	p, err := filepath.Abs(src)
	if err != nil {
		return err
	}

	sourceFileStat, err := os.Stat(p)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", p)
	}

	source, err := os.Open(p)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err = io.Copy(destination, source); err != nil {
		return err
	}

	return destination.Sync()
}

func CreateSparseFile(p string, size int64) error {
	file, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create sparse file failed: %w", err)
	}
	defer file.Close()

	if err := file.Truncate(size); err != nil {
		return fmt.Errorf("truncate sparse file failed: %w", err)
	}

	return file.Sync()
}
