// Copyright 2026 The gVisor Authors.
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

package tcp

import "testing"

// TestAdvertisedMSS pins the MSS invariant: whether the route is an ordinary
// one (MTU already net of the IPv4 header, 1380) or an IPIP one (MTU is the
// whole inner packet budget, 1400), the advertised MSS must be 1360 for
// InnerMTU=1400. The IPIP branch subtracts the 20-byte IPv4 header so a full
// segment does not exceed the inner budget and get re-split.
func TestAdvertisedMSS(t *testing.T) {
	tests := []struct {
		name     string
		routeMTU uint32
		needIPIP bool
		user     uint16
		want     uint16
	}{
		{name: "ordinary route network MTU", routeMTU: 1380, user: 0, want: 1360},
		{name: "ipip route inner budget", routeMTU: 1400, needIPIP: true, user: 0, want: 1360},
		{name: "user mss below max wins", routeMTU: 1380, user: 1200, want: 1200},
		{name: "user mss above max ignored", routeMTU: 1400, needIPIP: true, user: 1460, want: 1360},
	}
	for _, tc := range tests {
		if got := advertisedMSS(tc.routeMTU, tc.needIPIP, tc.user); got != tc.want {
			t.Errorf("%s: advertisedMSS(%d, %v, %d) = %d, want %d",
				tc.name, tc.routeMTU, tc.needIPIP, tc.user, got, tc.want)
		}
	}
}
