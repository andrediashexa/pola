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

func TestIPv4EroSubobjectSerializeLoose(t *testing.T) {
	so, err := NewIPv4EroSubObject(table.NewSegmentRSVPIPv4(netip.MustParseAddr("10.0.0.1"), true))
	assert.NoError(t, err)
	b, err := so.Serialize()
	assert.NoError(t, err)
	// L=1, type=1 -> 0x81 ; length 8 ; 10.0.0.1 ; prefix 32 ; reserved 0
	assert.Equal(t, []uint8{0x81, 0x08, 10, 0, 0, 1, 32, 0}, b)
}

func TestIPv4EroSubobjectSerializeStrict(t *testing.T) {
	so, err := NewIPv4EroSubObject(table.NewSegmentRSVPIPv4(netip.MustParseAddr("10.26.10.14"), false))
	assert.NoError(t, err)
	b, err := so.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, uint8(0x01), b[0]) // L=0, type=1
	assert.Equal(t, uint8(0x08), b[1])
}

func TestIPv4EroSubobjectRoundTrip(t *testing.T) {
	in := []uint8{0x81, 0x08, 10, 26, 10, 14, 32, 0}
	so := &IPv4EroSubobject{}
	assert.NoError(t, so.DecodeFromBytes(in))
	assert.True(t, so.LFlag)
	assert.Equal(t, "10.26.10.14", so.Addr.String())
	assert.Equal(t, uint8(32), so.PrefixLength)
	seg, ok := so.ToSegment().(table.SegmentRSVPIPv4)
	assert.True(t, ok)
	assert.Equal(t, "10.26.10.14", seg.Addr.String())
	assert.True(t, seg.Loose)
}

func TestEroObjectRSVPRoundTrip(t *testing.T) {
	segs := []table.Segment{
		table.NewSegmentRSVPIPv4(netip.MustParseAddr("10.26.10.17"), true),
		table.NewSegmentRSVPIPv4(netip.MustParseAddr("10.26.10.14"), true),
	}
	ero, err := NewEroObject(segs)
	assert.NoError(t, err)
	b, err := ero.Serialize()
	assert.NoError(t, err)

	decoded := &EroObject{}
	// skip the 4-byte common object header
	assert.NoError(t, decoded.DecodeFromBytes(ObjectTypeEROExplicitRoute, b[4:]))
	assert.Len(t, decoded.EroSubobjects, 2)
	sl := decoded.ToSegmentList()
	assert.Equal(t, "10.26.10.17", sl[0].SidString())
	assert.Equal(t, "10.26.10.14", sl[1].SidString())
}
