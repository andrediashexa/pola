// Copyright (c) 2025 NTT Communications Corporation
//
// This software is released under the MIT License.
// see https://github.com/nttcom/pola/blob/main/LICENSE

package server

import (
	"net/netip"
	"testing"

	pb "github.com/nttcom/pola/api/pola/v1"
	"github.com/nttcom/pola/pkg/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The gRPC CreateSRPolicy path (disable-path-compute) carries RSVP-TE hops via
// the "rsvp:<ipv4>" SID convention; buildSegmentList must turn them into
// SegmentRSVPIPv4 (loose) so the session emits an RSVP-TE PCInitiate.
func TestBuildSegmentListRSVP(t *testing.T) {
	req := &pb.CreateSRPolicyRequest{
		SrPolicy: &pb.SRPolicy{
			SrcAddr: netip.MustParseAddr("10.26.10.9").AsSlice(),
			DstAddr: netip.MustParseAddr("170.150.220.254").AsSlice(),
			SegmentList: []*pb.Segment{
				{Sid: "rsvp:10.26.10.17"},
				{Sid: "rsvp:10.26.10.14"},
			},
		},
	}

	segs, src, dst, err := buildSegmentList(&APIServer{}, req, true)
	require.NoError(t, err)
	assert.Equal(t, "10.26.10.9", src.String())
	assert.Equal(t, "170.150.220.254", dst.String())
	require.Len(t, segs, 2)
	for _, s := range segs {
		v, ok := s.(table.SegmentRSVPIPv4)
		require.True(t, ok, "segment must be RSVP IPv4")
		assert.True(t, v.Loose)
	}
	assert.Equal(t, "10.26.10.17", segs[0].SidString())
	assert.Equal(t, "10.26.10.14", segs[1].SidString())
}
