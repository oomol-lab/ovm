// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oomol-lab/ovm/pkg/utils"
	"golang.org/x/sync/errgroup"
)

type Context struct {
	Name           string
	VersionPath    string
	LogPath        string
	SocketPath     string
	IsCliMode      bool
	LockFile       string
	ExecutablePath string
	BindPID        int

	Endpoint          string
	SSHPort           int
	SSHKeyPath        string
	SSHPrivateKeyPath string
	SSHPublicKeyPath  string
	SSHPublicKey      string

	ForwardSocketPath     string
	SocketNetworkPath     string
	SocketInitrdVSockPath string
	SocketReadyPath       string

	CPUS         uint
	MemoryBytes  uint64
	KernelPath   string
	InitrdPath   string
	RootfsPath   string
	TargetPath   string
	DiskDataPath string
	DiskTmpPath  string

	cleanups []func() error
}

func Init() *Context {
	return &Context{}
}

func (c *Context) PreSetup() error {
	g := errgroup.Group{}

	g.Go(c.basic)
	g.Go(c.logPath)

	return g.Wait()
}

func (c *Context) Setup() (cleanup func() error, err error) {
	g := errgroup.Group{}

	g.Go(c.socketPath)
	g.Go(c.ssh)
	g.Go(c.sshPort)
	g.Go(c.target)

	return func() error {
		l := len(c.cleanups)
		if l == 0 {
			return nil
		}

		errs := make([]error, 0, l)
		wg := sync.WaitGroup{}

		wg.Add(l)
		for _, cleanup := range c.cleanups {
			go func(cleanup func() error) {
				defer wg.Done()
				if err := cleanup(); err != nil {
					errs = append(errs, err)
				}
			}(cleanup)
		}

		wg.Wait()

		if len(errs) > 0 {
			return fmt.Errorf("cleanup error: %v", errs)
		}

		return nil
	}, g.Wait()
}

func (c *Context) basic() error {
	c.Name = name
	c.CPUS = cpus
	c.MemoryBytes = memory * 1024 * 1024
	c.IsCliMode = cliMode
	c.BindPID = bindPID

	if err := os.MkdirAll("/tmp/ovm", 0755); err != nil {
		return err
	}

	if p, err := os.Executable(); err != nil {
		return fmt.Errorf("get executable path error: %w", err)
	} else {
		p, err := filepath.EvalSymlinks(p)
		if err != nil {
			return fmt.Errorf("eval symlink error: %w", err)
		}

		c.ExecutablePath = strings.ToLower(p)

		sum := md5.Sum([]byte(c.ExecutablePath))
		hash := hex.EncodeToString(sum[:])
		c.LockFile = "/tmp/ovm/" + hash + "-" + name + ".pid"
	}

	return nil
}

func (c *Context) socketPath() error {
	p, err := filepath.Abs(socketPath)
	if err != nil {
		return err
	}

	c.SocketPath = p
	c.ForwardSocketPath = path.Join(p, name+"-podman.sock")
	c.SocketNetworkPath = path.Join(p, name+"-vfkit-network.sock")
	c.SocketInitrdVSockPath = path.Join(p, name+"-initrd-vsock.sock")
	c.SocketReadyPath = path.Join(p, name+"-ready.sock")

	c.Endpoint = "unix://" + c.SocketNetworkPath

	if err := os.RemoveAll(c.SocketPath); err != nil {
		return err
	}

	if err := os.MkdirAll(c.SocketPath, 0755); err != nil {
		return err
	}

	c.cleanups = append(c.cleanups, func() error {
		return os.RemoveAll(c.SocketPath)
	})

	return nil
}

func (c *Context) ssh() error {
	p, err := filepath.Abs(sshKeyPath)
	if err != nil {
		return err
	}

	c.SSHKeyPath = p
	c.SSHPrivateKeyPath = path.Join(p, name)
	c.SSHPublicKeyPath = path.Join(p, name+".pub")

	if err := os.MkdirAll(p, 0700); err != nil {
		return err
	}

	{
		g := errgroup.Group{}
		g.Go(func() error {
			_, err := os.Stat(c.SSHPrivateKeyPath)
			return err
		})
		g.Go(func() error {
			_, err := os.Stat(c.SSHPublicKeyPath)
			return err
		})
		if err := g.Wait(); err != nil {
			_ = os.RemoveAll(c.SSHPrivateKeyPath)
			_ = os.RemoveAll(c.SSHPublicKeyPath)
			if err := utils.GenerateSSHKey(c.SSHKeyPath, name); err != nil {
				return err
			}
		}
	}

	{
		f, err := os.Open(c.SSHPublicKeyPath)
		if err != nil {
			return err
		}
		defer f.Close()

		b, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		c.SSHPublicKey = strings.TrimSpace(string(b))
	}

	return nil
}

func (c *Context) sshPort() error {
	port, err := utils.FindUsablePort(2233)
	if err != nil {
		return err
	}

	c.SSHPort = port

	return nil
}

func (c *Context) logPath() error {
	p, err := filepath.Abs(logPath)
	if err != nil {
		return err
	}

	c.LogPath = p

	return os.MkdirAll(c.LogPath, 0755)
}

func (c *Context) target() error {
	p, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	c.TargetPath = p
	if err := os.MkdirAll(c.TargetPath, 0755); err != nil {
		return err
	}

	c.VersionPath = path.Join(c.TargetPath, "version.json")
	c.KernelPath = path.Join(c.TargetPath, filepath.Base(kernelPath))
	c.InitrdPath = path.Join(c.TargetPath, filepath.Base(initrdPath))
	c.RootfsPath = path.Join(c.TargetPath, filepath.Base(rootfsPath))
	c.DiskDataPath = path.Join(c.TargetPath, "data.img")
	c.DiskTmpPath = path.Join(c.TargetPath, "tmp.img")

	{
		v := newVersion(c.TargetPath, c.VersionPath, c.DiskDataPath)
		if err := v.parseWithCmd(); err != nil {
			return err
		}

		if err := v.copy(); err != nil {
			return err
		}
	}

	if _, err := os.Stat(c.DiskTmpPath); err != nil {
		if err := utils.CreateSparseFile(c.DiskTmpPath, 1*1024*1024*1024*1024); err != nil {
			return err
		}
	}

	return nil
}
