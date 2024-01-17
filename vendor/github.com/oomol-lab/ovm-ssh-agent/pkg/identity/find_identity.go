/*
 * SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
 * SPDX-License-Identifier: MPL-2.0
 */

package identity

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/oomol-lab/ovm-ssh-agent/pkg/types"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var ignoreFiles = []string{
	"authorized_keys",
	"known_hosts",
	"known_hosts.old",
	"config",
	".DS_Store",
	"allowed_signers",
}

func isExecAny(mode os.FileMode) bool {
	return mode&0111 != 0
}

func IsPrivateKey(result *[]agent.AddedKey) func(path string, info os.FileInfo, err error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		// ignore large files
		if info.Size() > 1024*50 {
			return nil
		}

		// ignore executable files
		if isExecAny(info.Mode()) {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".pub") {
			return nil
		}

		for _, ignore := range ignoreFiles {
			if info.Name() == ignore {
				return nil
			}
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if key, err := ssh.ParseRawPrivateKey(content); err != nil {
			return nil
		} else {
			*result = append(*result, agent.AddedKey{
				PrivateKey: key,
			})
		}

		return nil
	}
}

func FindAll(log types.Logger) []agent.AddedKey {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Warnf("get user home dir failed: %v", err)
		return nil
	}

	sshDir := filepath.Join(dirname, ".ssh")

	state, err := os.Stat(sshDir)
	if err != nil {
		log.Warnf("stat ssh dir failed: %v", err)
		return nil
	}

	if !state.IsDir() {
		log.Warnf(".ssh dir is not a dir")
		return nil
	}

	result := make([]agent.AddedKey, 0, 20)
	if err := filepath.Walk(sshDir, IsPrivateKey(&result)); err != nil {
		log.Warnf("walk ssh dir failed: %v", err)
		return nil
	}

	return result
}
