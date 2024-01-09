// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package event

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Code-Hex/go-infinity-channel"
	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/logger"
	"golang.org/x/sync/errgroup"
)

type Name string

var (
	Initializing     Name = "Initializing"
	GVProxyReady     Name = "GVProxyReady"
	IgnitionProgress Name = "IgnitionProgress"
	IgnitionDone     Name = "IgnitionDone"
	VMReady          Name = "VMReady"
	Exit             Name = "Exit"
	Error            Name = "Error"
)

type datum struct {
	name    Name
	message string
}

type event struct {
	client  *http.Client
	log     *logger.Context
	channel *infinity.Channel[*datum]
}

var e *event

func Init(opt *cli.Context) error {
	log, err := logger.New(opt.LogPath, opt.Name+"-event")
	if err != nil {
		return err
	}

	if opt.EventSocketPath == "" {
		log.Info("no socket path, event will not be sent")
		return nil
	}

	c := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", opt.EventSocketPath)
			},
		},
		Timeout: 200 * time.Millisecond,
	}

	e = &event{
		client:  c,
		log:     log,
		channel: infinity.NewChannel[*datum](),
	}

	return nil
}

func Subscribe(g *errgroup.Group) {
	if e == nil {
		return
	}

	g.Go(func() error {
		for datum := range e.channel.Out() {
			uri := fmt.Sprintf("http://ovm/notify?event=%s&message=%s", datum.name, url.QueryEscape(datum.message))
			e.log.Infof("notify %s event to %s", datum.name, uri)

			if resp, err := e.client.Get(uri); err != nil {
				e.log.Warnf("notify %+v event failed: %v", *datum, err)
			} else {
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					e.log.Warnf("notify %+v event failed, status code is: %d", *datum, resp.StatusCode)
				}
			}

			if datum.name == Exit {
				e.channel.Close()
				e = nil
				return nil
			}
		}

		return nil
	})
}

func Notify(name Name) {
	if e == nil {
		return
	}

	e.channel.In() <- &datum{
		name: name,
	}
}

func NotifyError(err error) {
	if e == nil {
		return
	}

	e.channel.In() <- &datum{
		name:    Error,
		message: err.Error(),
	}
}
