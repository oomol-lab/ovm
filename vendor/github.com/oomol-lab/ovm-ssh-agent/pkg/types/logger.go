/*
 * SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
 * SPDX-License-Identifier: MPL-2.0
 */

package types

// Logger is the interface for logging.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
}
