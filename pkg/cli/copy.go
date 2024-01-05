// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/oomol-lab/ovm/pkg/utils"
	"github.com/oomol-lab/ovm/pkg/version"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type versionContext struct {
	kernel  string
	initrd  string
	rootfs  string
	dataImg string

	targetPath      string
	versionJSONPath string
	dataImgPath     string
}

func newVersion(targetPath, versionJSONPath, dataImgPath string) *versionContext {
	return &versionContext{
		targetPath:      targetPath,
		versionJSONPath: versionJSONPath,
		dataImgPath:     dataImgPath,
	}
}

func (c *versionContext) parseWithCmd() error {
	s := strings.Split(versions, ",")

	for _, val := range s {
		item := strings.Split(strings.TrimSpace(val), "=")
		switch strings.TrimSpace(item[0]) {
		case "kernel":
			c.kernel = strings.TrimSpace(item[1])
		case "initrd":
			c.initrd = strings.TrimSpace(item[1])
		case "rootfs":
			c.rootfs = strings.TrimSpace(item[1])
		case "dataImg":
			c.dataImg = strings.TrimSpace(item[1])
		default:
			continue
		}
	}

	if c.kernel == "" || c.initrd == "" || c.rootfs == "" || c.dataImg == "" {
		return errors.New("need kernel, initrd, rootfs, dataImg in versions")
	}

	return nil
}

func (c *versionContext) copy() error {
	g := errgroup.Group{}
	v := version.New(c.versionJSONPath)
	if err := v.Read(); err != nil {
		return err
	}

	{
		if v.HasUpdate("kernel", c.kernel) {
			addCopyTask(&g, kernelPath, c.targetPath)
		}
		v.Kernel = c.kernel
	}
	{
		if v.HasUpdate("initrd", c.initrd) {
			addCopyTask(&g, initrdPath, c.targetPath)
		}
		v.Initrd = c.initrd
	}
	{
		if v.HasUpdate("rootfs", c.rootfs) {
			addCopyTask(&g, rootfsPath, c.targetPath)
		}
		v.Rootfs = c.rootfs
	}
	{
		if v.HasUpdate("data_img", c.dataImg) {
			addCreateSparseTask(&g, c.dataImgPath, c.targetPath)
		}
		v.DataImg = c.dataImg
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return v.Write()
}

func addCopyTask(g *errgroup.Group, srcPath, targetPath string) {
	g.Go(func() error {
		return utils.Copy(srcPath, path.Join(targetPath, filepath.Base(srcPath)))
	})
}

func addCreateSparseTask(g *errgroup.Group, srcPath, targetPath string) {
	g.Go(func() error {
		p := path.Join(targetPath, filepath.Base(srcPath))
		if err := os.RemoveAll(p); err != nil {
			return err
		}

		return utils.CreateSparseFile(p, 8*1024*1024*1024*1024)
	})
}
