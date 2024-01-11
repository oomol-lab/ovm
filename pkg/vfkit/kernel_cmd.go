// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package vfkit

import (
	"strings"

	"github.com/oomol-lab/ovm/internal/consts"
	"github.com/oomol-lab/ovm/pkg/cli"
)

func kernelCMD(opt *cli.Context) string {
	sb := strings.Builder{}

	sb.Grow(300)

	// record Kernel and Systemd logs to console
	sb.WriteString("console=hvc0 ")

	// disable the creation of useless network interfaces.
	// see: https://github.com/oomol-lab/ovm-js/pull/23
	sb.WriteString("fb_tunnels=none ")

	// When a Mac wakes up from sleep, the hardware clock in the guest will lag for a while. This causes the kernel to think that the TSC is unstable, and thus switches to HPET.
	// However, the HPET is much slower than the TSC, causing any program involved with time-related code to experience a drop in performance.
	// Don't worry about any side effects of this option. In PR #19, we forced an update of the system time and hardware time in the guest.
	// In arm64, the clocksource is fixed as arch_sys_counter, so this issue does not exist.
	if consts.IsAMD64 {
		sb.WriteString("clocksource=tsc tsc=reliable ")
	}

	// systemd configuration
	// see: https://www.freedesktop.org/software/systemd/man/latest/systemd.html#Options%20that%20duplicate%20kernel%20command%20line%20settings
	{
		if !opt.IsCliMode {
			// don't colorize log output when not in cli mode
			sb.WriteString("systemd.log_color=false ")
		}

		// record Systemd targets logs to console
		sb.WriteString("systemd.default_standard_output=journal+console ")
		sb.WriteString("systemd.default_standard_error=journal+console ")
	}

	// enable debug logs
	if opt.IsCliMode {
		sb.WriteString("debug ")
	}

	return strings.TrimRight(sb.String(), " ")
}
