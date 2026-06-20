// Copyright (c) 2025 NTT Communications Corporation
//
// This software is released under the MIT License.
// see https://github.com/nttcom/pola/blob/main/LICENSE

package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSegmentRSVP(t *testing.T) {
	seg, err := NewSegment("rsvp:10.26.10.17")
	assert.NoError(t, err)
	v, ok := seg.(SegmentRSVPIPv4)
	assert.True(t, ok)
	assert.True(t, v.Loose)
	assert.Equal(t, "10.26.10.17", v.SidString())

	seg, err = NewSegment("rsvp-strict:10.26.10.14")
	assert.NoError(t, err)
	v, ok = seg.(SegmentRSVPIPv4)
	assert.True(t, ok)
	assert.False(t, v.Loose)

	_, err = NewSegment("rsvp:not-an-ip")
	assert.Error(t, err)
	_, err = NewSegment("rsvp:2001:db8::1") // IPv6 not valid for RSVP IPv4 ERO
	assert.Error(t, err)
}

func TestNewSegmentStillParsesSRandSRv6(t *testing.T) {
	seg, err := NewSegment("16001")
	assert.NoError(t, err)
	_, ok := seg.(SegmentSRMPLS)
	assert.True(t, ok)

	seg, err = NewSegment("2001:db8::1")
	assert.NoError(t, err)
	_, ok = seg.(SegmentSRv6)
	assert.True(t, ok)
}
