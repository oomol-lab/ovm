// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"fmt"
	"os"
	"strings"
)

func LocalTZ() (string, error) {
	tzPath, err := os.Readlink("/etc/localtime")
	if err != nil {
		return "", fmt.Errorf("readlink /etc/localtime failed: %v", err)
	}

	return strings.TrimPrefix(tzPath, "/var/db/timezone/zoneinfo"), nil
}
