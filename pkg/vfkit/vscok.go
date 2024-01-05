// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package vfkit

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/oomol-lab/ovm/pkg/logger"
	"inet.af/tcpproxy"
)

func connectVsocks(vm *vz.VirtualMachine, devs []*config.VirtioVsock, log *logger.Context) (release func(), err error) {
	releases := make([]func(), 0, len(devs))
	for _, vsock := range devs {
		port := vsock.Port
		socketURL := vsock.SocketURL
		log.Infof("Exposing vsock port %d on %s", port, socketURL)
		if release, err := listenVsock(vm, port, socketURL); err != nil {
			return nil, fmt.Errorf("error exposing vsock port %d: %v", port, err)
		} else {
			releases = append(releases, release)
		}
	}

	return func() {
		for _, r := range releases {
			r()
		}
	}, nil
}

// listenVsock proxies connections from a vsock port to a host unix socket.
// This allows the guest to initiate connections to the host over vsock
func listenVsock(vm *vz.VirtualMachine, port uint, vsockPath string) (release func(), err error) {
	var proxy tcpproxy.Proxy
	// listen for connections on the vsock port
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		parsed, err := url.Parse(laddr)
		if err != nil {
			return nil, err
		}
		switch parsed.Scheme {
		case "vsock":
			port, err := strconv.Atoi(parsed.Port())
			if err != nil {
				return nil, err
			}
			socketDevices := vm.SocketDevices()
			if len(socketDevices) != 1 {
				return nil, fmt.Errorf("VM has too many/not enough virtio-vsock devices (%d)", len(socketDevices))
			}
			return socketDevices[0].Listen(uint32(port))
		default:
			return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
		}
	}

	proxy.AddRoute(fmt.Sprintf("vsock://:%d", port), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("unix:%s", vsockPath),
		// when there's a connection to the vsock listener, connect to the provided unix socket
		DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, e error) {
			parsed, err := url.Parse(addr)
			if err != nil {
				return nil, err
			}
			switch parsed.Scheme {
			case "unix":
				var d net.Dialer
				return d.DialContext(ctx, parsed.Scheme, parsed.Path)
			default:
				return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
			}
		},
	})

	return func() {
		_ = proxy.Close()
	}, proxy.Start()
}
