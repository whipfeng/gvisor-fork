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

// TestAdvertisedMSS pins the MSS invariant: an ordinary route's packet budget is
// already net of the IPv4 header (1380), while an explicit inner-packet budget
// covers the whole inner IP packet (1400) whether the egress is a direct
// LinkNIC or IPIP. Both must advertise 1360 for InnerMTU=1400, so a full
// segment neither exceeds the inner budget nor is re-split at the encapsulation
// boundary.
func TestAdvertisedMSS(t *testing.T) {
	tests := []struct {
		name         string
		packetBudget uint32
		innerBudget  bool
		user         uint16
		want         uint16
	}{
		{name: "ordinary route network MTU", packetBudget: 1380, user: 0, want: 1360},
		{name: "inner budget direct LinkNIC", packetBudget: 1400, innerBudget: true, user: 0, want: 1360},
		{name: "inner budget IPIP", packetBudget: 1400, innerBudget: true, user: 0, want: 1360},
		{name: "user mss below max wins", packetBudget: 1380, user: 1200, want: 1200},
		{name: "user mss above max ignored", packetBudget: 1400, innerBudget: true, user: 1460, want: 1360},
	}
	for _, tc := range tests {
		if got := advertisedMSS(tc.packetBudget, tc.innerBudget, tc.user); got != tc.want {
			t.Errorf("%s: advertisedMSS(%d, %v, %d) = %d, want %d",
				tc.name, tc.packetBudget, tc.innerBudget, tc.user, got, tc.want)
		}
	}
}
