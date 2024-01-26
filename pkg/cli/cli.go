// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"flag"
	"fmt"
)

var (
	name            string
	logPath         string
	socketPath      string
	sshKeyPath      string
	cpus            uint
	memory          uint64
	kernelPath      string
	initrdPath      string
	rootfsPath      string
	targetPath      string
	versions        string
	eventSocketPath string
	cliMode         bool
	bindPID         int
	powerSaveMode   bool
	kernelDebug     bool
)

func Parse() {
	flag.StringVar(&name, "name", "", "Name of the virtual machine")
	flag.StringVar(&logPath, "log-path", "", "Directory to store logs")
	flag.StringVar(&socketPath, "socket-path", "", "Store all socket files")
	flag.StringVar(&sshKeyPath, "ssh-key-path", "", "Store SSH public and private keys")
	flag.UintVar(&cpus, "cpus", 0, "Number of CPUs")
	flag.Uint64Var(&memory, "memory", 0, "Amount of memory in megabytes")
	flag.StringVar(&kernelPath, "kernel-path", "", "Path to kernel image")
	flag.StringVar(&initrdPath, "initrd-path", "", "Path to initrd image")
	flag.StringVar(&rootfsPath, "rootfs-path", "", "Path to rootfs image")
	flag.StringVar(&targetPath, "target-path", "", "Store disk images and kernel/initrd/rootfs files")
	flag.StringVar(&versions, "versions", "", "Set version")
	flag.StringVar(&eventSocketPath, "event-socket-path", "", "Send event to this socket")
	flag.BoolVar(&cliMode, "cli", false, "Run in CLI mode")
	flag.IntVar(&bindPID, "bind-pid", 0, "OVM will exit when the bound pid exited")
	flag.BoolVar(&powerSaveMode, "power-save-mode", false, "Enable power save mode")
	flag.BoolVar(&kernelDebug, "kernel-debug", false, "Enable kernel debug")

	flag.Parse()

}

func Validate() error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if logPath == "" {
		return fmt.Errorf("log-path is required")
	}
	if socketPath == "" {
		return fmt.Errorf("socket-path is required")
	}
	if sshKeyPath == "" {
		return fmt.Errorf("ssh-key-path is required")
	}
	if cpus == 0 {
		return fmt.Errorf("vcpu is required")
	}
	if memory == 0 {
		return fmt.Errorf("memory is required")
	}
	if kernelPath == "" {
		return fmt.Errorf("kernel-path is required")
	}
	if initrdPath == "" {
		return fmt.Errorf("initrd-path is required")
	}
	if rootfsPath == "" {
		return fmt.Errorf("rootfs-path is required")
	}
	if targetPath == "" {
		return fmt.Errorf("disk-path is required")
	}
	if versions == "" {
		return fmt.Errorf("versions is required")
	}
	return nil
}
