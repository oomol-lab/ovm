// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"fmt"
	"net"
)

func portOccupied(port int) error {
	a, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("resolve TCPAddr failed, %w", err)
	}

	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return fmt.Errorf("port %d is occupied, %w", port, err)
	}

	defer l.Close()
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
