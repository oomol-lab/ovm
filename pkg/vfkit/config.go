// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package vfkit

import (
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/logger"
)

func vmConfig(opt *cli.Context, log *logger.Context) (*config.VirtualMachine, error) {
	bootloaderCMD := []string{"linux", "kernel=" + opt.KernelPath, "initrd=" + opt.InitrdPath, "cmdline=" + kernelCMD(opt)}
	log.Infof("bootloader params: %+v", bootloaderCMD)

	bootloader, err := config.BootloaderFromCmdLine(bootloaderCMD)
	if err != nil {
		return nil, err
	}

	log.Infof("vm cpu: %d, memory: %d", opt.CPUS, opt.MemoryBytes/1024/1024)

	vm := config.NewVirtualMachine(opt.CPUS, opt.MemoryBytes, bootloader)

	// Order cannot be disrupted
	{
		log.Infof("block devices: vda: '%s', vdb: '%s', vdc: '%s'", opt.RootfsPath, opt.DiskTmpPath, opt.DiskDataPath)

		rootfs, _ := config.VirtioBlkNew(opt.RootfsPath)
		_ = vm.AddDevice(rootfs) // vda

		tmp, _ := config.VirtioBlkNew(opt.DiskTmpPath)
		_ = vm.AddDevice(tmp) // vdb

		data, _ := config.VirtioBlkNew(opt.DiskDataPath)
		_ = vm.AddDevice(data) // vdc
	}

	{
		log.Infof("vsock device: network: '%d-%s', initrd: '%d-%s', ready: '%d-%s'", 1024, opt.SocketNetworkPath, 1025, opt.SocketInitrdVSockPath, 1026, opt.SocketReadyPath)

		network, _ := config.VirtioVsockNew(1024, opt.SocketNetworkPath, false)
		_ = vm.AddDevice(network) // vm network device

		initrd, _ := config.VirtioVsockNew(1025, opt.SocketInitrdVSockPath, false)
		_ = vm.AddDevice(initrd) // initrd vsock device (https://github.com/oomol-lab/vsock-guest-exec)

		ready, _ := config.VirtioVsockNew(1026, opt.SocketReadyPath, false)
		_ = vm.AddDevice(ready) // vm is ready (https://github.com/oomol-lab/ovm-core/blob/7c85e7603da0873099c1a288be1f70e44e24c1f5/buildroot_external/board/ovm/ready/rootfs-overlay/etc/systemd/system/ready.service)
	}

	if opt.IsCliMode {
		serial, _ := config.VirtioSerialNewStdio()
		_ = vm.AddDevice(serial) // serial device (output to stdio)
	} else {
		logPath, err := logger.NewWithoutStream(opt.LogPath, opt.Name+"-vm")
		if err != nil {
			log.Errorf("create serial logger error: %v", err)
			return nil, err
		}
		serial, _ := config.VirtioSerialNew(logPath)
		_ = vm.AddDevice(serial) // serial device (output to log file)
	}

	{
		log.Infof("mount devices: %+v", mounts.list)
		for _, dev := range mounts.toVFKit() {
			_ = vm.AddDevice(dev)
		}
	}

	rng, _ := config.VirtioRngNew()
	_ = vm.AddDevice(rng) // rng device (https://github.com/oomol-lab/ovm-js/pull/36)

	return vm, nil
}
