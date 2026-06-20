// Copyright (c) 2025 NTT Communications Corporation
//
// This software is released under the MIT License.
// see https://github.com/nttcom/pola/blob/main/LICENSE

package pcep

import (
	"errors"
	"net/netip"

	"github.com/nttcom/pola/pkg/table"
)

// IPv4 Prefix ERO Subobject — RFC 3209 §4.3.3.1 (RSVP-TE explicit route).
//
//	 0               1               2               3
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|L|   Type=1   |    Length=8   |          IPv4 address ...     |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	| ... IPv4 address (cont.)      | Prefix Length |   Reserved    |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
// L=1 means the hop is LOOSE (the PCC's CSPF expands it). On a Huawei NE8000 a
// STRICT loopback ERO fails RSVP signalling (Error 24/5), so RSVP-TE steering
// from a PCE must use loose hops.
const SubObjectTypeEROIPv4Prefix SubObjectType = 0x01

const ipv4EroSubobjectLength uint8 = 8

type IPv4EroSubobject struct {
	LFlag        bool
	Addr         netip.Addr
	PrefixLength uint8
}

func (so *IPv4EroSubobject) DecodeFromBytes(b []uint8) error {
	if len(b) < int(ipv4EroSubobjectLength) {
		return errors.New("IPv4 ERO subobject is too short")
	}
	so.LFlag = (b[0] & 0x80) != 0
	addr, ok := netip.AddrFromSlice(b[2:6])
	if !ok {
		return errors.New("invalid IPv4 address in ERO subobject")
	}
	so.Addr = addr
	so.PrefixLength = b[6]
	return nil
}

func (so *IPv4EroSubobject) Len() (uint16, error) {
	return uint16(ipv4EroSubobjectLength), nil
}

func (so *IPv4EroSubobject) Serialize() ([]uint8, error) {
	if !so.Addr.Is4() {
		return nil, errors.New("IPv4 ERO subobject requires an IPv4 address")
	}
	buf := make([]uint8, ipv4EroSubobjectLength)
	t := uint8(SubObjectTypeEROIPv4Prefix)
	if so.LFlag {
		t |= 0x80 // L-flag = loose
	}
	buf[0] = t
	buf[1] = ipv4EroSubobjectLength
	a4 := so.Addr.As4()
	copy(buf[2:6], a4[:])
	pfx := so.PrefixLength
	if pfx == 0 {
		pfx = 32
	}
	buf[6] = pfx
	buf[7] = 0 // reserved
	return buf, nil
}

func (so *IPv4EroSubobject) ToSegment() table.Segment {
	return table.SegmentRSVPIPv4{Addr: so.Addr, Loose: so.LFlag}
}

func NewIPv4EroSubObject(seg table.SegmentRSVPIPv4) (*IPv4EroSubobject, error) {
	if !seg.Addr.Is4() {
		return nil, errors.New("RSVP-TE IPv4 ERO requires an IPv4 address")
	}
	return &IPv4EroSubobject{
		LFlag:        seg.Loose,
		Addr:         seg.Addr,
		PrefixLength: 32,
	}, nil
}
