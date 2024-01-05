// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"
	"net"
	"time"
)

func AcceptTimeout(ctx context.Context, nl net.Listener, timeout <-chan time.Time) (conn net.Conn, err error) {
	acc := make(chan error, 1)
	defer close(acc)

	go func() {
		accept, err := nl.Accept()
		conn = accept
		acc <- err
	}()

	select {
	case <-ctx.Done():
		_ = nl.Close()

		return conn, fmt.Errorf("cancel wait net accept %s because ctx done", nl.Addr().String())
	case <-timeout:
		_ = nl.Close()

		return conn, fmt.Errorf("wait net accept timeout %s", nl.Addr().String())
	case err := <-acc:
		return conn, err
	}
}
