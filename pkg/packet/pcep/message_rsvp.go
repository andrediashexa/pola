// Copyright (c) 2025 NTT Communications Corporation
//
// This software is released under the MIT License.
// see https://github.com/nttcom/pola/blob/main/LICENSE

package pcep

import (
	"net/netip"

	"github.com/nttcom/pola/pkg/table"
)

// rsvpSegments adapts a typed RSVP hop list to the generic []table.Segment the
// ERO/SRP builders consume.
func rsvpSegments(hops []table.SegmentRSVPIPv4) []table.Segment {
	segs := make([]table.Segment, len(hops))
	for i, h := range hops {
		segs[i] = h
	}
	return segs
}

// NewPCInitiateMessageRSVP builds a PCInitiate (RFC 8281) for an RSVP-TE LSP:
//   - SRP with PATH-SETUP-TYPE = RSVP-TE (0)
//   - LSP object (symbolic name)
//   - ENDPOINTS object (source/destination)
//   - ERO of IPv4 prefix subobjects (RFC 3209)
//
// Unlike the SR-policy PCInitiate, it carries NO color/preference ASSOCIATION or
// VENDOR-INFORMATION objects (those are SR-policy concepts). Hops should be
// LOOSE — a Huawei NE8000 rejects a strict loopback ERO (RSVP Error 24/5); the
// PCC's own CSPF expands loose hops.
func NewPCInitiateMessageRSVP(srpID uint32, lspName string, lspDelete bool, plspID uint32, hops []table.SegmentRSVPIPv4, srcAddr, dstAddr netip.Addr) (*PCInitiateMessage, error) {
	segs := rsvpSegments(hops)
	m := &PCInitiateMessage{}
	var err error

	if m.SrpObject, err = NewSrpObject(segs, srpID, lspDelete); err != nil {
		return nil, err
	}
	if lspDelete {
		if m.LSPObject, err = NewLSPObject(lspName, nil, plspID); err != nil {
			return nil, err
		}
		return m, nil
	}
	if m.LSPObject, err = NewLSPObject(lspName, nil, 0); err != nil {
		return nil, err
	}
	if m.EndpointsObject, err = NewEndpointsObject(dstAddr, srcAddr); err != nil {
		return nil, err
	}
	if m.EroObject, err = NewEroObject(segs); err != nil {
		return m, err
	}
	return m, nil
}

// NewPCUpdMessageRSVP builds a PCUpd (RFC 8231) that re-routes a delegated
// RSVP-TE LSP through the given loose IPv4 hops. The generic PCUpd already emits
// SRP+LSP+ERO; with RSVP segments the SRP carries PST=0 and the ERO IPv4
// subobjects. Use to steer an existing active-delegate tunnel.
func NewPCUpdMessageRSVP(srpID uint32, lspName string, plspID uint32, hops []table.SegmentRSVPIPv4) (*PCUpdMessage, error) {
	return NewPCUpdMessage(srpID, lspName, plspID, rsvpSegments(hops))
}
