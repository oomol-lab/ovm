// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package server

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
	PodmanSocketPath string `json:"podmanSocketPath"`
}

type Server struct {
	vz  *vz.VirtualMachine
	vmC *config.VirtualMachine
	log *logger.Context
	opt *cli.Context
}

func New(vz *vz.VirtualMachine, vmC *config.VirtualMachine, log *logger.Context, opt *cli.Context) *Server {
	return &Server{
		vz:  vz,
		vmC: vmC,
		log: log,
		opt: opt,
	}
}

func (s *Server) mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(s.info())
	})
	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/requestStop", func(w http.ResponseWriter, r *http.Request) {
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

	return mux
}

func (s *Server) Start(ctx context.Context, g *errgroup.Group, nl net.Listener) {
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

func (s *Server) info() *infoResponse {
	return &infoResponse{
		PodmanSocketPath: s.opt.ForwardSocketPath,
	}
}

func (s *Server) state() *stateResponse {
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

func (s *Server) pause() error {
	s.log.Info("request /pause")
	err := s.vz.Pause()
	if err != nil {
		s.log.Warnf("request pause VM failed: %v", err)
	}

	return err
}

func (s *Server) resume() error {
	s.log.Info("request /resume")
	err := s.vz.Resume()
	if err != nil {
		s.log.Warnf("request resume VM failed: %v", err)
	}

	return err
}

func (s *Server) requestStop() error {
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

func (s *Server) stop() error {
	s.log.Info("request /stop")
	err := s.vz.Stop()
	if err != nil {
		s.log.Warnf("request stop VM failed: %v", err)
	}

	return err
}
