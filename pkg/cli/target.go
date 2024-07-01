// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/oomol-lab/ovm/pkg/utils"
	"golang.org/x/sync/errgroup"
)

type versionsJSON struct {
	Kernel string `json:"kernel"`
	Initrd string `json:"initrd"`
	Rootfs string `json:"rootfs"`
	Data   string `json:"data"`

	path           string
	needUpdateJSON bool
}

func newVersionsJSON(path string) (*versionsJSON, error) {
	v := &versionsJSON{
		path: path,
	}

	if err := parseVersions(); err != nil {
		return nil, err
	}

	if err := v.read(); err != nil {
		return nil, err
	}

	return v, nil
}

// read reads the versions file.
// If parsing fails, the file will be deleted.
func (v *versionsJSON) read() error {
	data, err := os.ReadFile(v.path)
	if err != nil {
		return os.RemoveAll(v.path)
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return os.RemoveAll(v.path)
	}

	return nil
}

func (v *versionsJSON) saveToDisk() error {
	if !v.needUpdateJSON {
		return nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return os.WriteFile(v.path, data, 0644)
}

func (v *versionsJSON) get(key string) string {
	switch key {
	case "kernel":
		return v.Kernel
	case "initrd":
		return v.Initrd
	case "rootfs":
		return v.Rootfs
	case "data":
		return v.Data
	default:
		return ""
	}
}

func (v *versionsJSON) set(key, val string) {
	var vK *string
	switch key {
	case "kernel":
		vK = &v.Kernel
	case "initrd":
		vK = &v.Initrd
	case "rootfs":
		vK = &v.Rootfs
	case "data":
		vK = &v.Data
	}

	if *vK != val {
		*vK = val
		v.needUpdateJSON = true
	}
}

type srcPath struct {
	key string
	p   string
}

type targetContext struct {
	targetPath string

	srcPaths []srcPath

	versionsJSON *versionsJSON
}

func newTarget(targetPath, kernelPath, initrdPath, rootfsPath, dataPath, versionsPath string) (*targetContext, error) {
	versionsJSON, err := newVersionsJSON(versionsPath)
	if err != nil {
		return nil, err
	}

	return &targetContext{
		targetPath: targetPath,
		srcPaths: []srcPath{
			{"kernel", kernelPath},
			{"initrd", initrdPath},
			{"rootfs", rootfsPath},
			{"data", dataPath},
		},

		versionsJSON: versionsJSON,
	}, nil
}

func (t *targetContext) handle() error {
	g := errgroup.Group{}

	for _, src := range t.srcPaths {
		distPath := path.Join(t.targetPath, filepath.Base(src.p))

		if exists, _ := utils.PathExists(distPath); !exists {
			t.copyOrCreate(src, &g)
			continue
		}

		if v := t.versionsJSON.get(src.key); v != versionsParams[src.key] {
			t.copyOrCreate(src, &g)
			continue
		}
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return t.versionsJSON.saveToDisk()
}

func (t *targetContext) copyOrCreate(src srcPath, g *errgroup.Group) {
	t.versionsJSON.set(src.key, versionsParams[src.key])
	distPath := path.Join(t.targetPath, filepath.Base(src.p))

	g.Go(func() error {
		if src.key == "data" {
			if err := os.RemoveAll(distPath); err != nil {
				return err
			}

			return utils.CreateSparseFile(distPath, 8*1024*1024*1024*1024)
		}

		return utils.Copy(src.p, distPath)
	})
}

var versionsParams = map[string]string{
	"kernel": "",
	"initrd": "",
	"rootfs": "",
	"data":   "",
}

func parseVersions() error {
	s := strings.Split(versions, ",")

	for _, val := range s {
		item := strings.Split(strings.TrimSpace(val), "=")
		if len(item) != 2 {
			continue
		}

		key := strings.TrimSpace(item[0])

		if _, ok := versionsParams[key]; !ok {
			continue
		}

		versionsParams[key] = strings.TrimSpace(item[1])
	}

	for name, v := range versionsParams {
		if v == "" {
			return fmt.Errorf("need %s in versions", name)
		}
	}

	return nil
}
