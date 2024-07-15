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
)

type key string

const (
	kApp   key = "app"
	kError key = "error"
	kExit  key = "exit"
)

type app string

const (
	Initializing     app = "Initializing"
	GVProxyReady     app = "GVProxyReady"
	IgnitionProgress app = "IgnitionProgress"
	IgnitionDone     app = "IgnitionDone"
	Ready            app = "Ready"
)

type datum struct {
	name    key
	message string
}

type event struct {
	client  *http.Client
	log     *logger.Context
	channel *infinity.Channel[*datum]
}

var e *event

// see: https://github.com/Code-Hex/go-infinity-channel/issues/1
var waitDone = make(chan struct{})

func Setup(opt *cli.Context) error {
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

	go func() {
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

			if datum.name == kExit {
				waitDone <- struct{}{}
				return
			}
		}
	}()

	return nil
}

func NotifyApp(name app) {
	if e == nil {
		return
	}

	e.channel.In() <- &datum{
		name:    kApp,
		message: string(name),
	}
}

func NotifyError(err error) {
	if e == nil {
		return
	}

	e.channel.In() <- &datum{
		name:    kError,
		message: err.Error(),
	}
}

func NotifyExit() {
	if e == nil {
		return
	}

	e.channel.In() <- &datum{
		name:    kExit,
		message: "",
	}

	<-waitDone
	close(waitDone)
	e.channel.Close()
}
