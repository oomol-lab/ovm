// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"fmt"
	"net"
	"strconv"
)

func portOccupied(port int) error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return fmt.Errorf("port %d is occupied, %v", port, err)
	}
	defer ln.Close()
	return nil
}

func FindUsablePort(startPort int) (int, error) {
	port := startPort
	maxPort := startPort + 100

	var lastErr error

	for port < maxPort {
		if err := portOccupied(port); err != nil {
			lastErr = err
		} else {
			return port, nil
		}
		port++
	}

	return 0, lastErr
}
