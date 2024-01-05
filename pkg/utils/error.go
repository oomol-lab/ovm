// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package utils

func ErrChan(err error) chan error {
	errCh := make(chan error, 1)
	errCh <- err
	return errCh
}
