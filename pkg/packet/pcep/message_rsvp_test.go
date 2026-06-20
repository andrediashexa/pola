// Copyright (c) 2025 NTT Communications Corporation
//
// This software is released under the MIT License.
// see https://github.com/nttcom/pola/blob/main/LICENSE

package pcep

import (
	"net/netip"
	"testing"

	"github.com/nttcom/pola/pkg/table"
	"github.com/stretchr/testify/assert"
)

func looseHops(addrs ...string) []table.SegmentRSVPIPv4 {
	hops := make([]table.SegmentRSVPIPv4, len(addrs))
	for i, a := range addrs {
		hops[i] = table.NewSegmentRSVPIPv4(netip.MustParseAddr(a), true)
	}
	return hops
}

func TestNewPCInitiateMessageRSVP(t *testing.T) {
	src := netip.MustParseAddr("10.26.10.9")
	dst := netip.MustParseAddr("170.150.220.254")
	hops := looseHops("10.26.10.17", "10.26.10.18", "10.26.10.14")

	m, err := NewPCInitiateMessageRSVP(1, "RSVP-LSP-1", false, 0, hops, src, dst)
	assert.NoError(t, err)

	// SRP carries an explicit PATH-SETUP-TYPE = RSVP-TE (0).
	var pst *PathSetupType
	for _, tlv := range m.SrpObject.TLVs {
		if v, ok := tlv.(*PathSetupType); ok {
			pst = v
		}
	}
	assert.NotNil(t, pst)
	assert.Equal(t, PathSetupTypeRSVPTE, pst.PathSetupType)

	// ENDPOINTS present; no SR-policy ASSOCIATION/VENDOR objects.
	assert.NotNil(t, m.EndpointsObject)
	assert.Nil(t, m.AssociationObject)
	assert.Nil(t, m.VendorInformationObject)

	// ERO = 3 loose IPv4 subobjects, in order.
	assert.Len(t, m.EroObject.EroSubobjects, 3)
	for _, so := range m.EroObject.EroSubobjects {
		ipv4, ok := so.(*IPv4EroSubobject)
		assert.True(t, ok)
		assert.True(t, ipv4.LFlag) // loose
	}
	sl := m.EroObject.ToSegmentList()
	assert.Equal(t, "10.26.10.17", sl[0].SidString())
	assert.Equal(t, "10.26.10.14", sl[2].SidString())

	// Whole message serializes.
	b, err := m.Serialize()
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestNewPCUpdMessageRSVP(t *testing.T) {
	m, err := NewPCUpdMessageRSVP(2, "RSVP-LSP-1", 42, looseHops("10.26.20.65", "170.150.220.254"))
	assert.NoError(t, err)
	assert.Equal(t, uint32(42), m.LSPObject.PlspID)
	assert.Len(t, m.EroObject.EroSubobjects, 2)
	b, err := m.Serialize()
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
}
