// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package sshagentsock

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/oomol-lab/ovm-ssh-agent/pkg/identity"
	"github.com/oomol-lab/ovm-ssh-agent/pkg/sshagent"
	"github.com/oomol-lab/ovm/pkg/logger"
)

var knownAgentPaths = []string{
	".1password/agent.sock",
}

func FindExtendedAgent() (socketPath string, ok bool) {
	if p, ok := os.LookupEnv("SSH_AUTH_SOCK"); ok {
		return p, true
	}

	home, err := os.UserHomeDir()
	if err != nil {
		goto LAUNCHD
	}

	for _, p := range knownAgentPaths {
		p = filepath.Join(home, p)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}

LAUNCHD:
	output, err := exec.Command("/bin/launchctl", "asuser", strconv.Itoa(os.Getuid()), "launchctl", "getenv", "SSH_AUTH_SOCK").CombinedOutput()
	if err != nil {
		return "", false
	}

	out := string(bytes.TrimSpace(output))
	if _, err := os.Stat(out); err == nil {
		return out, true
	}

	return "", false
}

func Start(sshAuthSocketPath string, log *logger.Context) (*sshagent.SSHAgent, error) {
	agent, err := sshagent.New(sshAuthSocketPath, log)
	if err != nil {
		log.Errorf("new ssh agent error: %v", err)
		return nil, err
	}

	keys := identity.FindAll(log)
	if err := agent.AddIdentities(keys...); err != nil {
		log.Errorf("add identities error: %v", err)
		return nil, err
	}

	if extendAgent, ok := FindExtendedAgent(); ok {
		log.Infof("found extended agent: %s", extendAgent)
		agent.SetExtendedAgent(extendAgent)
	}

	go func() {
		agent.Listen()
	}()

	return agent, nil
}
