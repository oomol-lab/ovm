// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"time"

	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/oomol-lab/ovm/pkg/utils"
)

func waitBindPID(ctx context.Context, log *logger.Context, pid int) {
	if pid == 0 {
		log.Info("pid is 0, no need to wait")
		<-ctx.Done()
		return
	}

	log.Infof("wait bind pid: %d exit", pid)

	for {
		if ctx.Err() != nil {
			log.Info("cancel wait bind pid, because context done")
			return
		}

		if !utils.ProcessExists(pid) {
			log.Infof("bind pid: %d exited", pid)
			return
		}

		time.Sleep(1 * time.Second)
	}
}
