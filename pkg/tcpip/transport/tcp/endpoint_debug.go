package tcp

import (
	"fmt"
	"sync"

	"gvisor.dev/gvisor/pkg/tcpip"
)

// debugEndpointRegistry tracks every live *Endpoint for diagnostics. It is
// package-level (not on protocol) to avoid touching save/restore state.
var debugEndpointRegistry sync.Map // *Endpoint -> struct{}

func registerDebugEndpoint(e *Endpoint) {
	debugEndpointRegistry.Store(e, struct{}{})
}

func unregisterDebugEndpoint(e *Endpoint) {
	debugEndpointRegistry.Delete(e)
}

// TCPFlowDebugInfo is a JSON-friendly snapshot of one endpoint's internals.
type TCPFlowDebugInfo struct {
	LocalAddr  string `json:"localAddr"`
	LocalPort  uint16 `json:"localPort"`
	RemoteAddr string `json:"remoteAddr"`
	RemotePort uint16 `json:"remotePort"`
	State      string `json:"state"`

	SndBufUsed int `json:"sndBufUsed"`
	SndBufSize int `json:"sndBufSize"`
	RcvBufUsed int `json:"rcvBufUsed"`

	HasSender     bool   `json:"hasSender"`
	CC            string `json:"cc"`
	Recovery      string `json:"recovery"`
	CongestState  string `json:"congestState"`
	SndUna        uint32 `json:"sndUna"`
	SndNxt        uint32 `json:"sndNxt"`
	Cwnd          int    `json:"cwnd"`
	Ssthresh      int    `json:"ssthresh"`
	Outstanding   int    `json:"outstanding"`
	SackedOut     int    `json:"sackedOut"`
	DupAcks       int    `json:"dupAcks"`
	SndWnd        uint32 `json:"sndWnd"`
	UndoValid     bool   `json:"undoValid"`
	UndoPending   bool   `json:"undoPending"`
	RTOus         int64  `json:"rtoUs"`
	SRTTus        int64  `json:"srttUs"`
	LastSendAgoMs int64  `json:"lastSendAgoMs"`
	UnsentBytes   int    `json:"unsentBytes"`
	WriteListSegs int    `json:"writeListSegs"`
	ResendTimer   string `json:"resendTimer"`
	ReorderTimer  string `json:"reorderTimer"`
	ProbeTimer    string `json:"probeTimer"`
	Corked        bool   `json:"corked"`
	ZeroWndProbe  bool   `json:"zeroWndProbe"`

	// RACK reordering diagnostics: ReoWnd is the allowance before RACK marks
	// a segment lost; MinRTT/ReoWndIncr drive it. Reord sticky means a state
	// where dup-ACK fast retransmit is intentionally disabled.
	ReoWndMs   int64 `json:"reoWndMs"`
	MinRTTus   int64 `json:"minRttUs"`
	RackRTTus  int64 `json:"rackRttUs"`
	ReoWndIncr int   `json:"reoWndIncr"`
	Reord      bool  `json:"reord"`
	DSACKSeen  bool  `json:"dsackSeen"`

	HasReceiver    bool   `json:"hasReceiver"`
	RcvNxt         uint32 `json:"rcvNxt"`
	RcvAcc         uint32 `json:"rcvAcc"`
	PendingBufUsed int    `json:"pendingBufUsed"`
	PendingRcvSegs int    `json:"pendingRcvSegs"`
}

func timerStateName(t timer) string {
	switch t.state {
	case timerUninitialized:
		return "uninit"
	case timerStateDisabled:
		return "disabled"
	case timerStateOrphaned:
		return "orphaned"
	case timerStateEnabled:
		return "enabled"
	default:
		return fmt.Sprintf("unknown(%d)", int(t.state))
	}
}

func congestionStateName(s tcpip.CongestionControlState) string {
	switch s {
	case tcpip.Open:
		return "open"
	case tcpip.RTORecovery:
		return "rto-recovery"
	case tcpip.FastRecovery:
		return "fast-recovery"
	case tcpip.SACKRecovery:
		return "sack-recovery"
	case tcpip.Disorder:
		return "disorder"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// DumpTCPFlowDebug snapshots all live endpoints matching filter. filter may
// be nil to dump everything. Best-effort diagnostic: individual endpoint
// locks are taken one at a time so a stuck endpoint stalls its own snapshot,
// not the whole dump.
func DumpTCPFlowDebug(filter func(local, remote tcpip.FullAddress) bool) []TCPFlowDebugInfo {
	var out []TCPFlowDebugInfo
	debugEndpointRegistry.Range(func(k, _ any) bool {
		e := k.(*Endpoint)
		e.mu.Lock()
		id := e.TransportEndpointInfo.ID
		local := tcpip.FullAddress{Addr: id.LocalAddress, Port: id.LocalPort}
		remote := tcpip.FullAddress{Addr: id.RemoteAddress, Port: id.RemotePort}
		if filter != nil && !filter(local, remote) {
			e.mu.Unlock()
			return true
		}

		info := TCPFlowDebugInfo{
			LocalAddr:  local.Addr.String(),
			LocalPort:  id.LocalPort,
			RemoteAddr: remote.Addr.String(),
			RemotePort: id.RemotePort,
			State:      e.EndpointState().String(),
		}

		e.sndQueueInfo.sndQueueMu.Lock()
		info.SndBufUsed = e.sndQueueInfo.SndBufUsed
		info.SndBufSize = e.sndQueueInfo.SndBufSize
		e.sndQueueInfo.sndQueueMu.Unlock()

		e.rcvQueueMu.Lock()
		info.RcvBufUsed = e.TCPRcvBufState.RcvBufUsed
		e.rcvQueueMu.Unlock()

		if snd := e.snd; snd != nil {
			info.HasSender = true
			info.CC = string(e.cc)
			info.Recovery = fmt.Sprintf("%T", snd.lr)
			info.CongestState = congestionStateName(snd.state)
			info.SndUna = uint32(snd.SndUna)
			info.SndNxt = uint32(snd.SndNxt)
			info.Cwnd = snd.SndCwnd
			info.Ssthresh = snd.Ssthresh
			info.Outstanding = snd.Outstanding
			info.SackedOut = snd.SackedOut
			info.DupAcks = snd.DupAckCount
			info.SndWnd = uint32(snd.SndWnd)
			info.UndoValid = snd.undoValid
			info.UndoPending = snd.undoPending
			info.RTOus = snd.RTO.Microseconds()
			info.SRTTus = snd.rtt.SRTT.Microseconds()
			if snd.LastSendTime != (tcpip.MonotonicTime{}) {
				info.LastSendAgoMs = e.stack.Clock().NowMonotonic().Sub(snd.LastSendTime).Milliseconds()
			}
			info.ResendTimer = timerStateName(snd.resendTimer)
			info.ReorderTimer = timerStateName(snd.reorderTimer)
			info.ProbeTimer = timerStateName(snd.probeTimer)
			info.Corked = snd.startCork
			info.ZeroWndProbe = snd.zeroWindowProbing
			for seg := snd.writeNext; seg != nil; seg = seg.Next() {
				info.UnsentBytes += seg.payloadSize()
				info.WriteListSegs++
			}
		}

		if rcv := e.rcv; rcv != nil {
			info.HasReceiver = true
			info.RcvNxt = uint32(rcv.RcvNxt)
			info.RcvAcc = uint32(rcv.RcvAcc)
			info.PendingBufUsed = rcv.PendingBufUsed
			info.PendingRcvSegs = rcv.pendingRcvdSegments.Len()
		}

		e.mu.Unlock()
		out = append(out, info)
		return true
	})
	return out
}
