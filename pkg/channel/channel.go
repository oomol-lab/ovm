// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package channel

import "github.com/Code-Hex/go-infinity-channel"

type _context struct {
	gvproxyReady chan bool
	vmReady      chan bool
	syncTime     *infinity.Channel[bool]
}

var c *_context

func init() {
	c = &_context{
		gvproxyReady: make(chan bool, 1),
		vmReady:      make(chan bool, 1),
		syncTime:     infinity.NewChannel[bool](),
	}
}

func Close() {
	close(c.gvproxyReady)
	close(c.vmReady)
	c.syncTime.Close()
}

func NotifyGVProxyReady() {
	c.gvproxyReady <- true
}

func ReceiveGVProxyReady() <-chan bool {
	return c.gvproxyReady
}

func NotifyVMReady() {
	c.vmReady <- true
}

func ReceiveVMReady() <-chan bool {
	return c.vmReady
}

func NotifySyncTime() {
	c.syncTime.In() <- true
}

func ReceiveSyncTime() <-chan bool {
	return c.syncTime.Out()
}
