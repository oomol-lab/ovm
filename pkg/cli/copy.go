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
		distPath := path.Join(targetPath, filepath.Base(kernelPath))
		if need, err := v.NeedCopy(distPath, "kernel", c.kernel); need {
			addCopyTask(&g, kernelPath, distPath)
		} else if err != nil {
			return err
		}
		v.Kernel = c.kernel
	}
	{
		distPath := path.Join(targetPath, filepath.Base(initrdPath))
		if need, err := v.NeedCopy(distPath, "initrd", c.initrd); need {
			addCopyTask(&g, initrdPath, distPath)
		} else if err != nil {
			return err
		}
		v.Initrd = c.initrd
	}
	{
		distPath := path.Join(targetPath, filepath.Base(rootfsPath))
		if need, err := v.NeedCopy(distPath, "rootfs", c.rootfs); need {
			addCopyTask(&g, rootfsPath, distPath)
		} else if err != nil {
			return err
		}
		v.Rootfs = c.rootfs
	}
	{
		distPath := path.Join(targetPath, filepath.Base(c.dataImgPath))
		if need, err := v.NeedCopy(distPath, "data_img", c.dataImg); need {
			addCreateSparseTask(&g, distPath)
		} else if err != nil {
			return err
		}
		v.DataImg = c.dataImg
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return v.Write()
}

func addCopyTask(g *errgroup.Group, srcPath, target string) {
	g.Go(func() error {
		return utils.Copy(srcPath, target)
	})
}

func addCreateSparseTask(g *errgroup.Group, target string) {
	g.Go(func() error {
		if err := os.RemoveAll(target); err != nil {
			return err
		}

		return utils.CreateSparseFile(target, 8*1024*1024*1024*1024)
	})
}
