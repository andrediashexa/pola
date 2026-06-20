// Copyright (c) 2025 NTT Communications Corporation
//
// This software is released under the MIT License.
// see https://github.com/nttcom/pola/blob/main/LICENSE

package table

import "net/netip"

// SegmentRSVPIPv4 is an RSVP-TE explicit-route hop: an IPv4 address that the
// PCC's RSVP signalling traverses. Loose=true emits the ERO subobject with the
// L-flag set (RFC 3209) — the PCC's own CSPF expands it (required by some
// implementations, e.g. Huawei NE8000, which rejects strict loopback EROs).
type SegmentRSVPIPv4 struct {
	Addr  netip.Addr
	Loose bool
}

func (seg SegmentRSVPIPv4) SidString() string {
	return seg.Addr.String()
}

func NewSegmentRSVPIPv4(addr netip.Addr, loose bool) SegmentRSVPIPv4 {
	return SegmentRSVPIPv4{Addr: addr, Loose: loose}
}
