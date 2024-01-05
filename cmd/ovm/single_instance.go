// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/oomol-lab/ovm/pkg/logger"
	"github.com/oomol-lab/ovm/pkg/pidlock"
	"github.com/oomol-lab/ovm/pkg/utils"
	"github.com/shirou/gopsutil/v3/process"
)

func makeSingleInstance(logPath string, lockFile, executablePath string) (lock *pidlock.Context, err error) {
	log, err := logger.NewWithoutManage(logPath, "single-instance")
	defer log.Close()

	if err != nil {
		return nil, fmt.Errorf("create single instance logger error: %w", err)
	}

	lock = pidlock.New(lockFile)

	if ok, err := utils.PathExists(lockFile); err != nil {
		return nil, fmt.Errorf("check pid file failed: %w", err)
	} else if !ok {
		return lock, lock.TryLock()
	}

	log.Info("pid lock file exists, try kill previous process")

	owner, err := lock.Owner()
	if err != nil {
		log.Warnf("get pid lock owner error: %v, try lock", err)

		return lock, lock.TryLock()
	}

	log.Infof("pid lock owner: %d", owner)

	proc, err := process.NewProcess(int32(owner))
	if err != nil {
		log.Infof("pid lock owner %d not exists, error: %v, try lock", owner, err)
		return lock, lock.TryLock()
	}

	exe, err := proc.Exe()
	if err != nil {
		log.Errorf("get pid lock owner %d exe error: %v, try lock", owner, err)
		return lock, lock.TryLock()
	}

	realExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		log.Errorf("get pid lock owner %d real path %s error: %v, try lock", owner, exe, err)
	}

	log.Infof("pid lock owner %d exe: '%s'", owner, realExe)

	if strings.ToLower(realExe) != executablePath {
		log.Infof("pid lock owner %d exe '%s' not match '%s', try lock", owner, realExe, executablePath)
		return lock, lock.TryLock()
	}

	if err := utils.NotifyProcessSuicide(owner); err != nil {
		log.Errorf("kill previous process error: %v, try force kill", err)

		if err := utils.ForceKill(owner); err != nil {
			log.Errorf("force kill previous process error: %v. try lock", err)

			return lock, lock.TryLock()
		}
	}

	log.Infof("send SIGTERM to %d success, wait 10s process exit", owner)

	processExited := false
	for i := 0; i < 10; i++ {
		if !utils.ProcessExists(owner) {
			processExited = true
			break
		}

		time.Sleep(1 * time.Second)
	}

	if !processExited {
		log.Warnf("process %d not exited, try force kill", owner)
		if err := utils.ForceKill(owner); err != nil {
			log.Errorf("force kill previous process error: %v, try lock", err)

			return lock, lock.TryLock()
		}
	}

	log.Info("kill previous process success, try lock again")

	return lock, lock.TryLock()
}
