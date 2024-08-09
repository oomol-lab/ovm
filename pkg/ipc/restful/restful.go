// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package restful

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/oomol-lab/ovm/pkg/cli"
	"github.com/oomol-lab/ovm/pkg/logger"
	"golang.org/x/sync/errgroup"
)

type stateResponse struct {
	State          string `json:"state"`
	CanStart       bool   `json:"canStart"`
	CanRequestStop bool   `json:"canRequestStop"`
	CanStop        bool   `json:"canStop"`
	CanPause       bool   `json:"canPause"`
	CanResume      bool   `json:"canResume"`
}

type infoResponse struct {
	PodmanSocketPath  string `json:"podmanSocketPath"`
	SSHPort           int    `json:"sshPort"`
	SSHUser           string `json:"sshUser"`
	SSHPublicKeyPath  string `json:"sshPublicKeyPath"`
	SSHPrivateKeyPath string `json:"sshPrivateKeyPath"`
	SSHPublicKey      string `json:"sshPublicKey"`
	SSHPrivateKey     string `json:"sshPrivateKey"`
}

type Restful struct {
	vz  *vz.VirtualMachine
	vmC *config.VirtualMachine
	log *logger.Context
	opt *cli.Context
}

func New(vz *vz.VirtualMachine, vmC *config.VirtualMachine, log *logger.Context, opt *cli.Context) *Restful {
	return &Restful{
		vz:  vz,
		vmC: vmC,
		log: log,
		opt: opt,
	}
}

type powerSaveModeBody struct {
	Enable bool `json:"enable"`
}

func (s *Restful) mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "get only", http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(s.info())
	})
	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "get only", http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(s.state())
	})
	mux.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "post only", http.StatusBadRequest)
			return
		}

		if err := s.pause(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "post only", http.StatusBadRequest)
			return
		}

		if err := s.resume(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/request-stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "post only", http.StatusBadRequest)
			return
		}

		if err := s.requestStop(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "post only", http.StatusBadRequest)
			return
		}

		if err := s.stop(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/power-save-mode", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "put only", http.StatusBadRequest)
			return
		}

		var body powerSaveModeBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			s.log.Warnf("Failed to decode request body: %v", err)
			http.Error(w, "failed to decode request body", http.StatusBadRequest)
			return
		}

		s.powerSaveMode(body.Enable)
	})

	return mux
}

func (s *Restful) Start(ctx context.Context, g *errgroup.Group, nl net.Listener) {
	g.Go(func() error {
		<-ctx.Done()
		return nl.Close()
	})

	g.Go(func() error {
		server := &http.Server{
			Handler: s.mux(),
		}
		return server.Serve(nl)
	})
}

func (s *Restful) info() *infoResponse {
	s.log.Info("request /info")
	return &infoResponse{
		PodmanSocketPath:  s.opt.ForwardSocketPath,
		SSHPort:           s.opt.SSHPort,
		SSHUser:           "root",
		SSHPublicKeyPath:  s.opt.SSHPublicKeyPath,
		SSHPrivateKeyPath: s.opt.SSHPrivateKeyPath,
		SSHPublicKey:      s.opt.SSHPublicKey,
		SSHPrivateKey:     s.opt.SSHPrivateKey,
	}
}

func (s *Restful) state() *stateResponse {
	s.log.Info("request /state")
	return &stateResponse{
		State:          s.vz.State().String(),
		CanStart:       s.vz.CanStart(),
		CanRequestStop: s.vz.CanRequestStop(),
		CanStop:        s.vz.CanStop(),
		CanPause:       s.vz.CanPause(),
		CanResume:      s.vz.CanResume(),
	}
}

func (s *Restful) pause() error {
	s.log.Info("request /pause")
	err := s.vz.Pause()
	if err != nil {
		s.log.Warnf("request pause VM failed: %v", err)
	}

	return err
}

func (s *Restful) resume() error {
	s.log.Info("request /resume")
	err := s.vz.Resume()
	if err != nil {
		s.log.Warnf("request resume VM failed: %v", err)
	}

	return err
}

func (s *Restful) requestStop() error {
	s.log.Info("request /requestStop")
	ok, err := s.vz.RequestStop()
	if err != nil {
		s.log.Warnf("request requestStop VM failed: %v", err)
	} else if !ok {
		err = fmt.Errorf("request requestStop VM failed, ok is false")
		s.log.Warnf("request requestStop VM failed: %v", err)
	}

	return err
}

func (s *Restful) stop() error {
	s.log.Info("request /stop")
	err := s.vz.Stop()
	if err != nil {
		s.log.Warnf("request stop VM failed: %v", err)
	}

	return err
}

func (s *Restful) powerSaveMode(enable bool) {
	s.log.Info("request /powerSaveMode")
	s.opt.PowerSaveMode = enable
}
