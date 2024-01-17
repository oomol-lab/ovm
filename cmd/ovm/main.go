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
	"github.com/oomol-lab/ovm/pkg/ipc/event"
	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/oomol-lab/ovm/pkg/sshagentsock"
	"github.com/oomol-lab/ovm/pkg/utils"
	"github.com/oomol-lab/ovm/pkg/vfkit"
	"golang.org/x/sync/errgroup"
)

var (
	opt    *cli.Context
	sigs   = make(chan os.Signal, 1)
	cleans []func()
)

func init() {
	cli.Parse()
	if err := cli.Validate(); err != nil {
		fmt.Printf("validate flags error: %v\n", err)
		exit(1)
	}

	opt = cli.Init()
	if err := opt.PreSetup(); err != nil {
		fmt.Printf("pre setup error: %v\n", err)
		exit(1)
	}
}

func main() {
	// See: https://github.com/crc-org/vfkit/pull/13/commits/906916ab9b92af7a5662fd7fe9246d61d39da4ee
	signal.Ignore(syscall.SIGPIPE)

	{
		if lock, err := makeSingleInstance(opt.LogPath, opt.LockFile, opt.ExecutablePath); err != nil {
			fmt.Println("make single instance error:", err)
			exit(1)
		} else {
			cleans = append(cleans, lock.Unlock)
		}
	}

	log, err := logger.New(opt.LogPath, opt.Name+"-ovm")
	if err != nil {
		fmt.Printf("create ovm logger error: %v\n", err)
		exit(1)
	}

	if err := opt.Setup(); err != nil {
		log.Errorf("setup error: %v", err)
		exit(1)
	}

	{
		if err := event.Init(opt); err != nil {
			log.Errorf("event init error: %v", err)
			exit(1)
		}

		g := errgroup.Group{}
		event.Subscribe(&g)
		cleans = append(cleans, func() {
			_ = g.Wait()
		})
	}

	agent, err := sshagentsock.Start(opt.SSHAuthSocketPath, log)
	if err != nil {
		log.Errorf("start ssh agent sock error: %v", err)
		exit(1)
	}

	event.Notify(event.Initializing)

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	if err := ready(ctx, g, opt, log); err != nil {
		log.Errorf("ready failed: %v", err)
		cancel()
		exit(1)
	}

	g.Go(func() error {
		<-ctx.Done()
		return agent.Close()
	})

	g.Go(func() error {
		waitBindPID(ctx, log, opt.BindPID)
		cancel()
		return nil
	})

	g.Go(func() error {
		return gvproxy.Run(ctx, g, opt)
	})

	g.Go(func() error {
		return vfkit.Run(ctx, g, opt)
	})

	g.Go(func() error {
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
		event.NotifyError(err)
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
			event.Notify(event.VMReady)
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
	event.Notify(event.Exit)
	for _, clean := range cleans {
		clean()
	}
	close(sigs)
	channel.Close()
	logger.CloseAll()
	os.Exit(exitCode)
}
