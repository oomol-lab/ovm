// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oomol-lab/ovm/pkg/channel"
	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/gvproxy"
	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/oomol-lab/ovm/pkg/utils"
	"github.com/oomol-lab/ovm/pkg/vfkit"
	"golang.org/x/sync/errgroup"
)

var opt *cli.Context

func init() {
	cli.Parse()
	opt = cli.Init()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// See: https://github.com/crc-org/vfkit/pull/13/commits/906916ab9b92af7a5662fd7fe9246d61d39da4ee
	signal.Ignore(syscall.SIGPIPE)

	if err := cli.Validate(); err != nil {
		fmt.Printf("validate flags error: %v\n", err)
		exit(1)
	}

	cleanup, err := opt.Setup()
	if err != nil {
		fmt.Printf("setup error: %v\n", err)
		exit(1)
	}

	log, err := logger.New(opt.LogPath, opt.Name+"-ovm")
	if err != nil {
		fmt.Printf("create ovm logger error: %v\n", err)
		exit(1)
	}

	g.Go(func() error {
		<-ctx.Done()
		log.Info("cli setup cleanup...")

		if err := cleanup(); err != nil {
			log.Errorf("cleanup failed: %v", err)
		}

		log.Info("cli setup cleanup done")

		return nil
	})

	if err := ready(ctx, g, opt, log); err != nil {
		log.Errorf("ready failed: %v", err)
		cancel()
		exit(1)
	}

	g.Go(func() error {
		return gvproxy.Run(ctx, g, opt)
	})

	g.Go(func() error {
		return vfkit.Run(ctx, g, opt)
	})

	g.Go(func() error {
		sigs := make(chan os.Signal, 1)
		defer close(sigs)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

		select {
		case sig := <-sigs:
			log.Warnf("received %s signal, exiting...", sig)
			cancel()
			return errors.New("signal caught")
		case <-ctx.Done():
			return nil
		}
	})

	if err := g.Wait(); err != nil {
		log.Errorf("main error: %v", err)
		exit(1)
	} else {
		log.Info("main exit")
		exit(0)
	}
}

func ready(ctx context.Context, g *errgroup.Group, opt *cli.Context, log *logger.Context) error {
	nl, err := net.Listen("unix", opt.SocketReadyPath)
	if err != nil {
		return err
	}

	g.Go(func() error {
		conn, err := utils.AcceptTimeout(ctx, nl, time.After(30*time.Second))
		if err != nil {
			log.Errorf("ready accept timeout: %v", err)
			return err
		}

		if _, rerr := bufio.NewReader(conn).ReadString('\n'); rerr != nil {
			log.Errorf("read ready failed: %v", rerr)
			err = rerr
		} else {
			channel.NotifyVMReady()
		}

		if cerr := conn.Close(); cerr != nil {
			log.Errorf("close ready connection failed: %v", cerr)
			err = cerr
		}

		return err
	})

	return nil
}

func exit(exitCode int) {
	channel.Close()
	logger.CloseAll()
	os.Exit(exitCode)
}
