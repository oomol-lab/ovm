/*
 * SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
 * SPDX-License-Identifier: MPL-2.0
 */

package sshagent

import (
	"net"

	"github.com/oomol-lab/ovm-ssh-agent/pkg/types"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type ProxyAgent struct {
	local              agent.Agent
	upstreamSocketPath string
	log                types.Logger
}

// NewProxyAgent creates a new ProxyAgent.
func NewProxyAgent(log types.Logger) *ProxyAgent {
	return &ProxyAgent{
		local: agent.NewKeyring(),
		log:   log,
	}
}

// GetExtendedAgentSocketPath returns the extended agent path.
func (a *ProxyAgent) GetExtendedAgentSocketPath() string {
	return a.upstreamSocketPath
}

// SetExtendedAgent sets the extended agent path.
func (a *ProxyAgent) SetExtendedAgent(socketPath string) {
	a.upstreamSocketPath = socketPath
}

// AddIdentities adds identities to the agent(local).
// It not adds identities to the extended agent.
func (a *ProxyAgent) AddIdentities(key ...agent.AddedKey) error {
	for _, k := range key {
		if err := a.Add(k); err != nil {
			return err
		}
	}

	return nil
}

func (a *ProxyAgent) refreshExtendedAgent() agent.ExtendedAgent {
	p := a.upstreamSocketPath
	if p == "" {
		return nil
	}

	conn, err := net.Dial("unix", p)
	if err != nil {
		a.log.Warnf("dial extended agent failed: %v", err)
		return nil
	}

	return agent.NewClient(conn)
}

// List returns the identities known to the agent(local + extended).
func (a *ProxyAgent) List() ([]*agent.Key, error) {
	l, err := a.local.List()

	if ea := a.refreshExtendedAgent(); ea != nil {
		us, err2 := ea.List()
		err = err2

		if err != nil {
			a.log.Warnf("get upstream list failed: %v", err)
		} else {
			l = append(l, us...)
		}
	}

	return l, err
}

// Add adds a private key to the agent(local).
// It will not add from extended agent.
func (a *ProxyAgent) Add(key agent.AddedKey) error {
	return a.local.Add(key)
}

// Remove removes identities with the given public key (local).
// It will not remove from extended agent.
func (a *ProxyAgent) Remove(key ssh.PublicKey) error {
	return a.local.Remove(key)
}

// RemoveAll removes all identities (local).
// It will not remove all from extended agent.
func (a *ProxyAgent) RemoveAll() error {
	return a.local.RemoveAll()
}

// Lock locks the agent (local + extended).
func (a *ProxyAgent) Lock(passphrase []byte) error {
	err := a.local.Lock(passphrase)
	if err != nil {
		a.log.Warnf("lock local agent failed: %v", err)
	}

	if ea := a.refreshExtendedAgent(); ea != nil {
		err = ea.Lock(passphrase)
	}

	return err
}

// Unlock undoes the effect of Lock (local + extended).
func (a *ProxyAgent) Unlock(passphrase []byte) error {
	err := a.local.Unlock(passphrase)
	if err != nil {
		a.log.Warnf("unlock local agent failed: %v", err)
	}

	if ea := a.refreshExtendedAgent(); ea != nil {
		err = ea.Unlock(passphrase)
	}

	return err
}

// Sign returns a signature by signing data with the given public key (local + extended).
// Prioritize signing from the local. If signing from the local source fails, then try extended.
func (a *ProxyAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	sig, err := a.local.Sign(key, data)
	if err == nil {
		return sig, nil
	}

	if ea := a.refreshExtendedAgent(); ea != nil {
		sig, err = ea.Sign(key, data)
	}

	return sig, err
}

// Signers returns signers for all signers (local + extended).
func (a *ProxyAgent) Signers() ([]ssh.Signer, error) {
	signers, err := a.local.Signers()
	if err != nil {
		a.log.Warnf("get local signers failed: %v", err)
	}

	if ea := a.refreshExtendedAgent(); ea != nil {
		us, err2 := ea.Signers()
		err = err2

		if err != nil {
			a.log.Warnf("get upstream signers failed: %v", err)
		} else {
			signers = append(signers, us...)
		}
	}

	return signers, err
}

// SignWithFlags returns a signature by signing data with the given public key (local + extended).
// Prioritize signing from the local. If signing from the local source fails, then try extended.
func (a *ProxyAgent) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	sig, err := a.local.(agent.ExtendedAgent).SignWithFlags(key, data, flags)
	if err == nil {
		return sig, nil
	}

	if ea := a.refreshExtendedAgent(); ea != nil {
		sig, err = ea.SignWithFlags(key, data, flags)
	}

	return sig, err
}

// Extension not supported.
func (a *ProxyAgent) Extension(extensionType string, contents []byte) ([]byte, error) {
	return nil, agent.ErrExtensionUnsupported
}
