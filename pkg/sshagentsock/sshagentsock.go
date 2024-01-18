// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package sshagentsock

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/oomol-lab/ovm-ssh-agent/pkg/identity"
	"github.com/oomol-lab/ovm-ssh-agent/pkg/sshagent"
	"github.com/oomol-lab/ovm/pkg/logger"
)

var knownAgentPaths = []string{
	".1password/agent.sock",
	"Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock",
}

// FindExtendedAgent finds the extended agent path.
// The find will be done in the following order:
//
// 1. Check if the environment variable SSH_AUTH_SOCK exists (if it is the built-in agent in macOS, it will be used as an alternative).
// 2. Check if any known third-party agent exists at the specified path.
// 3. Get the ssh auth sock of the current system using launchctl.
// 4. If all the above steps fail, use the alternative. Otherwise, return empty
func FindExtendedAgent() (socketPath string, ok bool) {
	if p, ok := os.LookupEnv("SSH_AUTH_SOCK"); ok {
		if strings.Contains(p, "com.apple.launchd.") {
			socketPath = p
		} else {
			return p, true
		}
	}

	if home, err := os.UserHomeDir(); err != nil {
		goto LAUNCHD
	} else {
		for _, p := range knownAgentPaths {
			p = filepath.Join(home, p)
			if _, err := os.Stat(p); err == nil {
				return p, true
			}
		}
	}

LAUNCHD:
	output, err := exec.Command("/bin/launchctl", "asuser", strconv.Itoa(os.Getuid()), "launchctl", "getenv", "SSH_AUTH_SOCK").CombinedOutput()
	if err == nil {
		out := string(bytes.TrimSpace(output))
		if _, err := os.Stat(out); err == nil {
			return out, true
		}
	}

	return socketPath, false
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
