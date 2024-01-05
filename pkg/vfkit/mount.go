// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package vfkit

import (
	"fmt"

	"github.com/crc-org/vfkit/pkg/config"
)

type fs struct {
	tag      string
	shareDir string
}

type _mounts struct {
	list []fs
}

var mounts = &_mounts{
	list: []fs{
		{
			tag:      "vfkit-share-user",
			shareDir: "/Users",
		},
		{
			tag:      "vfkit-share-var-folders",
			shareDir: "/var/folders",
		},
		{
			tag:      "vfkit-share-private",
			shareDir: "/private",
		},
	},
}

func (m *_mounts) toVFKit() (devices []config.VirtioDevice) {
	for _, fs := range m.list {
		d, _ := config.VirtioFsNew(fs.shareDir, fs.tag)
		devices = append(devices, d)
	}

	return devices
}

func (m *_mounts) toFSTAB() (result []string) {
	for _, fs := range m.list {
		fstab := fmt.Sprintf("%s %s virtiofs defaults 0 0", fs.tag, fs.shareDir)
		result = append(result, fstab)
	}

	return result
}
