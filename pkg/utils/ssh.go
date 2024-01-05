// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"fmt"
	"io"
	"os/exec"
	"path"
)

func GenerateSSHKey(p, name string) error {
	pn := path.Join(p, name)

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", pn, "-N", "")
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	waitErr := cmd.Wait()
	if waitErr == nil {
		return nil
	}

	errMsg, err := io.ReadAll(stdErr)
	if err != nil {
		return fmt.Errorf("key generation failed, unable to read from stderr: %w", waitErr)
	}

	return fmt.Errorf("failed to generate keys: %s: %w", string(errMsg), waitErr)
}
