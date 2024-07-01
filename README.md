# OVM

[![license]](https://github.com/oomol-lab/ovm/blob/main/LICENSE) [![repo size]](https://github.com/oomol-lab/ovm) [![release]](https://github.com/oomol-lab/ovm/releases/latest)

Run ovm-core virtual machine on Apple Virtualization Framework.

## Requirements

- macOS 12.3 or later
- Linux image must use [ovm core]

### Usage

Currently, we only provide the option to start via the command line.

### Command Line Parameters

#### `-name` (Required)

Name of the virtual machine.

#### `-cpus` (Required)

The number of CPUs allocated to the virtual machine.

#### `-memory` (Required)

The amount of memory allocated to the virtual machine (in MB).

#### `-log-path` (Required)

We request to provide a directory to store the logs of the latest 3 instances, for the purpose of troubleshooting.

The format of the log file name is as follows:

* ${name}-ovm.log (latest)
* ${name}-ovm.2.log
* ${name}-ovm.3.log
* ${name}-vfkit.log (latest)
* ${name}-vfkit.2.log
* ${name}-vfkit.3.log
* ...

#### `-socket-path` (Required)

During the startup process of the virtual machine, ovm will create some socket files. To facilitate management. Every time ovm starts, it will delete the files in the directory.

#### `-ssh-key-path` (Required)

Store SSH public and private keys. You can connect to the virtual machine through here the SSH public key.

Format: `${name}-ovm` and `${name}-ovm.pub`

#### `-kernel-path` (Required)

Path to the kernel image.

Regarding the `kernel` field, if the system is Mac ARM64 (M series), the kernel file needs to be uncompressed (not **bzImage**). For more information on this, please refer to: [kernel arm64 booting]

#### `-initrd-path` (Required)

Path to the initial ramdisk image

#### `-rootfs-path` (Required)

Path to rootfs image

#### `-target-path` (Required)

In order to address the issues that may occur when some files are damaged or other malfunctions happen, the program will first copy the files from the `kernel/initrd/rootfs` to this directory.

At the same time, ovm will also create `tmp.img` and data.img in this directory. Where `data.img` is the data (images, containers, etc.) of the virtual machine.

#### `-versions` (Required)

Set versions of the kernel/initrd/rootfs/data

Format: `kernel=version,initrd=version,rootfs=version,data=version`

When the version number differs from the previous one, the new file will be used to overwrite the previous file.

#### `-bind-pid` (Optional)

OVM will exit when the bound pid exited

#### `-power-save-mode` (Optional)

Enable power save mode.

Pause the guest when the Mac goes to sleep, resume the guest when the Mac wakes up, and synchronize the time.

#### `-event-socket-path` (Optional)

Send event to this socket.

When a socket file is passed to this parameter, the ovm sends the current status to this socket. The sent request is: `http://ovm/notify?event=EVENT&message=MESSAGE`

For more about this, please see: [ipc event]

#### `-cli` (Optional)

Run in CLI mode.

When this parameter is passed in, the debug parameter of the kernel will also be enabled (in order to display more detailed logs).

#### `-help` (Optional)

Show help message.

[license]: https://img.shields.io/github/license/oomol-lab/ovm?style=flat-square&color=9cf
[repo size]: https://img.shields.io/github/repo-size/oomol-lab/ovm?style=flat-square&color=9cf
[release]: https://img.shields.io/github/v/release/oomol-lab/ovm?style=flat-square&color=9cf
[ovm core]: https://github.com/oomol-lab/ovm-core
[kernel arm64 booting]: https://www.kernel.org/doc/Documentation/arm64/booting.txt
[ipc event]: https://github.com/oomol-lab/ovm/blob/285d338ccf36c4f584cc1ec6800d1164278353c5/pkg/ipc/event/event.go
