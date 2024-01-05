// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package channel

type _context struct {
	gvproxyReady chan bool
	vmReady      chan bool
}

var c *_context

func init() {
	c = &_context{
		gvproxyReady: make(chan bool, 1),
		vmReady:      make(chan bool, 1),
	}
}

func Close() {
	close(c.gvproxyReady)
	close(c.vmReady)
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
