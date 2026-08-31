// Copyright 2024 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kernel

import (
	"testing"

	"gvisor.dev/gvisor/pkg/abi/linux"
)

// TestInitSignalDiscarded verifies the discard rule that keeps a PID namespace's
// init process alive under Linux SIGNAL_UNKILLABLE semantics
// (kernel/signal.c:sig_task_ignored(), pid_namespaces(7)).
func TestInitSignalDiscarded(t *testing.T) {
	dflAct := linux.SigAction{Handler: linux.SIG_DFL}
	ignAct := linux.SigAction{Handler: linux.SIG_IGN}
	handlerAct := linux.SigAction{Handler: 0x1000} // arbitrary user handler

	for _, tc := range []struct {
		name   string
		sig    linux.Signal
		act    linux.SigAction
		forced bool
		want   bool
	}{
		// In-sandbox (forced == false): SIGNAL_UNKILLABLE semantics.
		// SIGKILL/SIGSTOP and default-fatal signals with no handler are
		// discarded, but an installed handler receives the signal.
		{"sigkill from peer", linux.SIGKILL, dflAct, false, true},
		{"sigstop from peer", linux.SIGSTOP, dflAct, false, true},
		{"sigterm default peer", linux.SIGTERM, dflAct, false, true},
		{"sigtstp default peer (stop)", linux.SIGTSTP, dflAct, false, true},
		{"sigterm handler peer", linux.SIGTERM, handlerAct, false, false},
		{"sigquit handler peer (core)", linux.SIGQUIT, handlerAct, false, false},
		{"sigterm ignored peer", linux.SIGTERM, ignAct, false, true},
		// Non-fatal-by-default signals are still always delivered.
		{"sigchld default peer", linux.SIGCHLD, dflAct, false, false},
		{"sigwinch handler peer", linux.SIGWINCH, handlerAct, false, false},

		// Out-of-sandbox (forced == true): Linux semantics.
		// SIGKILL/SIGSTOP are delivered so the control plane can tear the
		// sandbox down.
		{"sigkill forced", linux.SIGKILL, dflAct, true, false},
		{"sigstop forced", linux.SIGSTOP, dflAct, true, false},
		// Other fatal signals are force-delivered only to an installed handler;
		// with the default disposition they are discarded (Linux force-delivers
		// only sig_kernel_only signals).
		{"sigterm handler forced", linux.SIGTERM, handlerAct, true, false},
		{"sigterm default forced", linux.SIGTERM, dflAct, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := initSignalDiscarded(tc.sig, tc.act, tc.forced); got != tc.want {
				t.Errorf("initSignalDiscarded(%d, handler=%#x, forced=%t) = %t, want %t",
					tc.sig, tc.act.Handler, tc.forced, got, tc.want)
			}
		})
	}
}
