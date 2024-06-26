// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package vfkit

import (
	"fmt"
	"net"
	"time"

	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/ipc/event"
	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/oomol-lab/ovm/pkg/utils"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

func cmd(opt *cli.Context) (string, error) {
	localTZ, err := utils.LocalTZ()
	if err != nil {
		return "", err
	}
	tz := fmt.Sprintf("ln -sf /usr/share/zoneinfo/%s /mnt/overlay/etc/localtime; echo %s > /mnt/overlay/etc/timezone", localTZ, localTZ)

	fstab := ""
	for _, item := range mounts.toFSTAB() {
		fstab += item + "\\\\n"
	}

	mount := fmt.Sprintf("echo -e %s >> /mnt/overlay/etc/fstab", fstab)
	authorizedKeys := fmt.Sprintf("mkdir -p /mnt/overlay/root/.ssh; echo %s >> /mnt/overlay/root/.ssh/authorized_keys", opt.SSHPublicKey)
	ready := fmt.Sprintf("echo -e \"date -s @%d;\\\\necho Ready | socat -v -d -d - VSOCK-CONNECT:2:1026\" > /mnt/overlay/opt/ready.command", time.Now().Unix())

	return fmt.Sprintf("%s; %s; %s; %s", mount, authorizedKeys, ready, tz), nil
}

func ignition(ctx context.Context, g *errgroup.Group, opt *cli.Context, log *logger.Context) error {
	listen, err := net.Listen("unix", opt.SocketInitrdVSockPath)
	if err != nil {
		return fmt.Errorf("listen ignition socket failed: %v", err)
	}

	cmdStr, err := cmd(opt)
	if err != nil {
		return fmt.Errorf("generate ignition command failed: %v", err)
	}

	g.Go(func() error {
		conn, err := utils.AcceptTimeout(ctx, listen, time.After(15*time.Second))
		if err != nil {
			log.Errorf("ignition accept timeout: %v", err)
			return err
		}

		if _, werr := conn.Write([]byte(cmdStr)); werr != nil {
			log.Errorf("write ignition command failed: %v", werr)
			err = werr
		} else {
			log.Info("write ignition command success")
			event.NotifyApp(event.IgnitionDone)
		}

		if cerr := conn.Close(); cerr != nil {
			log.Errorf("close ignition connection failed: %v", cerr)
			err = cerr
		}

		return err
	})

	return nil
}
