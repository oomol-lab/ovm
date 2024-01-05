// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package version

import (
	"encoding/json"
	"os"
)

type List struct {
	Kernel      string `json:"kernel"`
	Initrd      string `json:"initrd"`
	Rootfs      string `json:"rootfs"`
	DataImg     string `json:"data_img"`
	versionPath string
}

func New(p string) *List {
	return &List{
		versionPath: p,
	}
}

func (l *List) Read() error {
	if _, err := os.Stat(l.versionPath); err != nil {
		return nil
	}

	data, err := os.ReadFile(l.versionPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &l)
}

func (l *List) Write() error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}

	return os.WriteFile(l.versionPath, data, 0644)
}

func (l *List) HasUpdate(t, v string) bool {
	var r string
	switch t {
	case "kernel":
		r = l.Kernel
	case "initrd":
		r = l.Initrd
	case "rootfs":
		r = l.Rootfs
	case "data_img":
		r = l.DataImg
	}

	return r != v
}
