// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"syscall"
)

func NotifyProcessSuicide(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

func ForceKill(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}

func ProcessExists(pid int) bool {
	return syscall.Kill(pid, syscall.Signal(0)) == nil
}
