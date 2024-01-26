// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package powermonitor

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/oomol-lab/ovm/pkg/channel"
	"github.com/oomol-lab/ovm/pkg/logger"
	"golang.org/x/sync/errgroup"
)

var (
	timeSyncConn *net.Conn
)

func initTimeSync(ctx context.Context, g *errgroup.Group, socketPath string, log *logger.Context) error {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen time sync socket file error: %w", err)
	}

	g.Go(func() error {
		<-ctx.Done()
		return listener.Close()
	})

	g.Go(func() error {
		log.Info("waiting for time sync connection")

		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept time sync socket file error: %w", err)
		}

		timeSyncConn = &conn
		log.Info("time sync connected")

		return nil
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				log.Info("cancel sync time event receive")
				return nil
			case <-channel.ReceiveSyncTime():
				log.Info("receive sync time event")
				break
			}

			if timeSyncConn == nil {
				continue
			}

			log.Info("start sync time")

			command := []byte(fmt.Sprintf("date -s @%d", time.Now().Unix()))
			length := len(command)
			header := make([]byte, 2)
			binary.LittleEndian.PutUint16(header, uint16(length))

			if err := writeConn(header); err != nil {
				return fmt.Errorf("write time sync header error: %w", err)
			}
			if err := writeConn(command); err != nil {
				return fmt.Errorf("write time sync command error: %w", err)
			}

			log.Info("sync time success")
		}
	})

	return nil
}

func writeConn(data []byte) error {
	total := 0
	for {
		now, err := (*timeSyncConn).Write(data[total:])
		if err != nil {
			return err
		}

		total += now
		if total == len(data) {
			return nil
		}
	}
}
