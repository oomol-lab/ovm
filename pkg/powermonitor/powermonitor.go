// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package powermonitor

import (
	"context"

	"github.com/Code-Hex/vz/v3"
	"github.com/oomol-lab/ovm/pkg/channel"
	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/prashantgupta24/mac-sleep-notifier/notifier"
	"golang.org/x/sync/errgroup"
)

func Setup(ctx context.Context, g *errgroup.Group, opt *cli.Context, vm *vz.VirtualMachine, log *logger.Context) error {
	if err := initTimeSync(ctx, g, opt.TimeSyncSocketPath, log); err != nil {
		return err
	}

	ch := notifier.GetInstance().Start()

	log.Info("power monitor started")

	g.Go(func() error {
		<-ctx.Done()
		log.Info("power monitor stopping")
		notifier.GetInstance().Quit()
		close(ch)
		log.Info("power monitor stopped")
		return nil
	})

	g.Go(func() error {
		for activity := range ch {

			log.Infof("os %s, power save mode: %v", activity.Type, opt.PowerSaveMode)

			switch activity.Type {
			case notifier.Awake:
				if !opt.PowerSaveMode {
					log.Info("not power save mode, notify sync time")
					channel.NotifySyncTime()
					continue
				}

				if !vm.CanResume() {
					log.Warnf("VM can not resume, current state: %s", vm.State())
					continue
				}

				if err := vm.Resume(); err != nil {
					log.Warnf("resume VM failed: %v", err)
				} else {
					log.Infof("resume VM success")
				}

			case notifier.Sleep:
				if !opt.PowerSaveMode {
					continue
				}

				if !vm.CanPause() {
					log.Warnf("VM can not pause, current state: %s", vm.State())
					continue
				}

				if err := vm.Pause(); err != nil {
					log.Warnf("pause VM failed: %v", err)
				} else {
					log.Infof("pause VM success")
				}
			}
		}

		log.Info("listen power monitor event exited")

		return nil
	})

	return nil
}
