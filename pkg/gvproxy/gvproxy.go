// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package gvproxy

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/containers/gvisor-tap-vsock/pkg/sshclient"
	"github.com/containers/gvisor-tap-vsock/pkg/transport"
	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"github.com/containers/gvisor-tap-vsock/pkg/virtualnetwork"
	"github.com/oomol-lab/ovm/pkg/channel"
	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/ipc/event"
	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const (
	gatewayIP   = "192.168.127.1"
	sshHostPort = "192.168.127.2:22"
	hostIP      = "192.168.127.254"
	host        = "host"
	gateway     = "gateway"
)

func Run(ctx context.Context, g *errgroup.Group, opt *cli.Context) error {
	log, err := logger.New(opt.LogPath, opt.Name+"-gvproxy")
	if err != nil {
		return fmt.Errorf("create gvproxy logger error: %v", err)
	}

	config := types.Configuration{
		Debug:             false,
		CaptureFile:       "",
		MTU:               5000,
		Subnet:            "192.168.127.0/24",
		GatewayIP:         gatewayIP,
		GatewayMacAddress: "5a:94:ef:e4:0c:dd",
		DHCPStaticLeases: map[string]string{
			"192.168.127.2": "5a:94:ef:e4:0c:ee",
		},
		DNS: []types.Zone{
			{
				Name: "containers.internal.",
				Records: []types.Record{
					{
						Name: gateway,
						IP:   net.ParseIP(gatewayIP),
					},
					{
						Name: host,
						IP:   net.ParseIP(hostIP),
					},
				},
			},
			{
				Name: "docker.internal.",
				Records: []types.Record{
					{
						Name: gateway,
						IP:   net.ParseIP(gatewayIP),
					},
					{
						Name: host,
						IP:   net.ParseIP(hostIP),
					},
				},
			},
		},
		DNSSearchDomains: searchDomains(log),
		Forwards: map[string]string{
			fmt.Sprintf("127.0.0.1:%d", opt.SSHPort): sshHostPort,
		},
		NAT: map[string]string{
			hostIP: "127.0.0.1",
		},
		GatewayVirtualIPs: []string{hostIP},
		VpnKitUUIDMacAddresses: map[string]string{
			"c3d68012-0208-11ea-9fd7-f2189899ab08": "5a:94:ef:e4:0c:ee",
		},
		Protocol: types.HyperKitProtocol,
	}
	vn, err := virtualnetwork.New(&config)
	if err != nil {
		return err
	}

	{
		log.Infof("listening %s", opt.Endpoint)
		ln, err := transport.Listen(opt.Endpoint)
		if err != nil {
			return errors.Wrap(err, "cannot listen")
		}
		httpServe(ctx, g, ln, vn.Mux())
	}

	ln, err := vn.Listen("tcp", fmt.Sprintf("%s:80", gatewayIP))
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/services/forwarder/all", vn.Mux())
	mux.Handle("/services/forwarder/expose", vn.Mux())
	mux.Handle("/services/forwarder/unexpose", vn.Mux())
	httpServe(ctx, g, ln, mux)

	channel.NotifyGVProxyReady()
	event.NotifyApp(event.GVProxyReady)

	g.Go(func() error {
		select {
		case <-ctx.Done():
			log.Info("skip create ssh forward, because context done")
			return nil
		case <-channel.ReceiveVMReady():
			log.Info("VM is ready, creating podman socket forward")
			break
		}

		src := &url.URL{
			Scheme: "unix",
			Path:   opt.ForwardSocketPath,
		}

		dest := &url.URL{
			Scheme: "ssh",
			User:   url.User("root"),
			Host:   sshHostPort,
			Path:   "/run/podman/podman.sock",
		}
		defer os.RemoveAll(opt.ForwardSocketPath)

		log.Infof("ssh private key path: %s", opt.SSHPrivateKeyPath)
		forward, err := sshclient.CreateSSHForward(ctx, src, dest, opt.SSHPrivateKeyPath, vn)
		if err != nil {
			return err
		}
		go func() {
			<-ctx.Done()
			forward.Close()
		}()

	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			default:
				// proceed
			}
			err := forward.AcceptAndTunnel(ctx)
			if err != nil {
				log.Infof("Error occurred handling ssh forwarded connection: %q", err)
			}
		}
		return nil
	})

	return nil
}

func searchDomains(log *logger.Context) []string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		log.Warnf("open /etc/resolv.conf file error: %v", err)
		return nil
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	searchPrefix := "search "
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), searchPrefix) {
			searchDomains := strings.Split(strings.TrimPrefix(sc.Text(), searchPrefix), " ")
			log.Warnf("Using search domains: %v", searchDomains)
			return searchDomains
		}
	}
	if err := sc.Err(); err != nil {
		log.Warnf("scan /etc/resolv.conf file error: %v", err)
		return nil
	}
	return nil
}

func httpServe(ctx context.Context, g *errgroup.Group, ln net.Listener, mux http.Handler) {
	g.Go(func() error {
		<-ctx.Done()
		return ln.Close()
	})
	g.Go(func() error {
		s := &http.Server{
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		return s.Serve(ln)
	})
}
