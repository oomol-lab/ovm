// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package vfkit

import (
	"strings"

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

	// systemd configuration
	// see: https://www.freedesktop.org/software/systemd/man/latest/systemd.html#Options%20that%20duplicate%20kernel%20command%20line%20settings
	{
		if !opt.IsCliMode {
			// don't colorize log output when not in cli mode
			sb.WriteString("systemd.log_color=false ")
		}

		// record Systemd targets logs to console
		sb.WriteString("systemd.default_standard_output=journal+console ")
		sb.WriteString("systemd.default_standard_error=journal+console debug")
	}

	return sb.String()
}
