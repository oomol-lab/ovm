// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package powermonitor

import (
	"context"
	"fmt"

	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/prashantgupta24/mac-sleep-notifier/notifier"
	"golang.org/x/sync/errgroup"
)

func Setup(ctx context.Context, g *errgroup.Group, opt *cli.Context, log *logger.Context) error {
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
			if activity.Type == notifier.Awake {
				log.Info("start sync time")
				if err := syncTime(); err != nil {
					return fmt.Errorf("sync time failed: %w", err)
				}
			}
		}

		log.Info("listen power monitor event exited")

		return nil
	})

	return nil
}
