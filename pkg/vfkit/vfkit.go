// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package vfkit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/vf"
	"github.com/oomol-lab/ovm/pkg/channel"
	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/ipc/event"
	"github.com/oomol-lab/ovm/pkg/ipc/restful"
	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/oomol-lab/ovm/pkg/powermonitor"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, g *errgroup.Group, opt *cli.Context) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for tag, dir := range opt.ExtendShareDir {
		mounts.extend(tag, dir)
	}

	log, err := logger.New(opt.LogPath, opt.Name+"-vfkit")
	if err != nil {
		return fmt.Errorf("create vfkit logger error: %v", err)
	}

	vmC, err := vmConfig(opt, log)
	if err != nil {
		log.Errorf("creating virtual machine config failed: %v", err)
		return err
	}

	vzVMConfig, err := vf.ToVzVirtualMachineConfig(vmC)
	if err != nil {
		log.Errorf("converting virtual machine config to vz failed: %v", err)
		return err
	}

	vm, err := vz.NewVirtualMachine(vzVMConfig)
	if err != nil {
		log.Errorf("creating vz virtual machine failed: %v", err)
		return err
	}

	{
		nl, err := net.Listen("unix", opt.RestfulSocketPath)
		if err != nil {
			log.Errorf("create server failed: %v", err)
			return err
		}
		restful.New(vm, vmC, log, opt).Start(ctx, g, nl)
	}

	select {
	case <-ctx.Done():
		log.Infof("skip start VM, because context done")
		return nil
	case <-time.After(10 * time.Second):
		msg := "timeout waiting for gvproxy to start"
		log.Error(msg)
		return errors.New(msg)
	case <-channel.ReceiveGVProxyReady():
		log.Info("gvproxy is ready, start VM")
		break
	}

	if err := powermonitor.Setup(ctx, g, opt, vm, log); err != nil {
		log.Errorf("setup powermonitor failed: %v", err)
		return err
	}

	vmState := make(chan vz.VirtualMachineState, 1)

	g.Go(func() error {
		for {
			state := <-vm.StateChangedNotify()
			log.Infof("VM state changed: %s", state)
			vmState <- state

			switch state {
			case vz.VirtualMachineStateStopped, vz.VirtualMachineStateError:
				log.Infof("stop listen VM state, because VM interruption, current state is: %s", state)
				return nil
			case vz.VirtualMachineStateResuming:
				channel.NotifySyncTime()
			default:
				// do nothing
			}
		}
	})

	if err := vm.Start(); err != nil {
		return err
	}

	event.NotifyApp(event.IgnitionProgress)

	if err := ignition(ctx, g, opt, log); err != nil {
		log.Errorf("ignition failed: %v", err)
		return err
	}

	if err := waitForVMState(vmState, vz.VirtualMachineStateRunning, time.After(5*time.Second)); err != nil {
		log.Errorf("waiting for VM to start failed: %v", err)
		return err
	}

	log.Infof("virtual machine is running")

	g.Go(func() error {
		devs := vmC.VirtioVsockDevices()
		release, err := connectVsocks(vm, devs, log)
		if err != nil {
			log.Errorf("connecting vsocks failed: %v", err)
			return err
		}
		log.Infof("vsocks are connected")

		<-ctx.Done()
		log.Infof("cancel listen vsocks, because context done")
		release()

		return nil
	})

	g.Go(func() error {
		if err := waitForVMState(vmState, vz.VirtualMachineStateStopped, nil); err != nil {
			log.Errorf("waiting for VM to stop failed: %v", err)
			return err
		}

		msg := "VM is stopped in waitForVMState"
		log.Warn(msg)
		return errors.New(msg)
	})

	g.Go(func() error {
		<-ctx.Done()
		log.Infof("stop VM, because context done")

		if err := stopVM(vm, log); err != nil {
			log.Errorf("error stopping VM: %v", err)
		} else {
			log.Infof("VM is stopped in stopVM")
		}

		return nil
	})

	return nil
}

func waitForVMState(chState <-chan vz.VirtualMachineState, state vz.VirtualMachineState, timeout <-chan time.Time) error {
	for {
		select {
		case newState := <-chState:
			if newState == state {
				return nil
			}
			if newState == vz.VirtualMachineStateError {
				return fmt.Errorf("VM state is error, expected state: %s", state)
			}
		case <-timeout:
			return fmt.Errorf("timeout waiting for VM %s", state)
		}
	}
}

func stopVM(vm *vz.VirtualMachine, log *logger.Context) error {
	err := requestStopVM(vm, log)
	if err == nil {
		return nil
	}

	log.Errorf("requesting VM to stop failed: %v", err)

	state := vm.State()
	if state == vz.VirtualMachineStateStopped || state == vz.VirtualMachineStateError {
		log.Infof("VM stopped, state is: %s", state)
		return nil
	}

	log.Infof("try to force stop VM, current state is: %s", state)

	if err := vm.Stop(); err != nil {
		log.Errorf("force stop VM failed: %v", err)
		return err
	}

	log.Infof("force stop VM succeeded")
	return nil
}

func requestStopVM(vm *vz.VirtualMachine, log *logger.Context) error {
	stateAlreadyStopping := false

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		if ctx.Err() != nil {
			return errors.New("timeout waiting for VM to stop")
		}

		switch vm.State() {
		case vz.VirtualMachineStateStopped:
			log.Infof("VM is already stopped")
			return nil

		case vz.VirtualMachineStateStopping:
			if !stateAlreadyStopping {
				log.Infof("VM state is stopping, waiting for it to stop")
				stateAlreadyStopping = true
			}

		case vz.VirtualMachineStateError:
			log.Errorf("VM is in error state in stopVM")
			return nil

		default:
			if vm.CanRequestStop() {
				log.Infof("requesting VM to stop")
				if ok, err := vm.RequestStop(); err != nil || !ok {
					if err != nil {
						log.Errorf("requesting VM to stop failed: %v. Forcing stop", err)
					} else {
						log.Errorf("requesting VM to stop: not ok. Forcing stop")
					}
				}
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}
