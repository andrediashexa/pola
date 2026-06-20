// Copyright (c) 2025 NTT Communications Corporation
//
// This software is released under the MIT License.
// see https://github.com/nttcom/pola/blob/main/LICENSE

package server

import (
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/nttcom/pola/pkg/packet/pcep"
	"github.com/nttcom/pola/pkg/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// loopbackConns returns a connected pair of *net.TCPConn (server, client).
func loopbackConns(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	type res struct {
		c   net.Conn
		err error
	}
	ch := make(chan res, 1)
	go func() {
		c, err := ln.Accept()
		ch <- res{c, err}
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	r := <-ch
	require.NoError(t, r.err)
	return r.c.(*net.TCPConn), client.(*net.TCPConn)
}

func rsvpPolicy() table.SRPolicy {
	return table.SRPolicy{
		Name:    "RSVP-SESSION-TEST",
		SrcAddr: netip.MustParseAddr("10.26.10.9"),
		DstAddr: netip.MustParseAddr("170.150.220.254"),
		SegmentList: []table.Segment{
			table.NewSegmentRSVPIPv4(netip.MustParseAddr("10.26.10.17"), true),
			table.NewSegmentRSVPIPv4(netip.MustParseAddr("10.26.10.14"), true),
		},
	}
}

// SendPCInitiate with an RSVP segment list must put a well-formed RSVP-TE
// PCInitiate on the wire (the exact bytes the RSVP builder produces) and bump
// the SRP-ID. This exercises the session->wire path end to end over real TCP.
func TestSessionSendPCInitiateRSVP(t *testing.T) {
	srv, cli := loopbackConns(t)
	defer srv.Close()
	defer cli.Close()

	ss := NewSession(0, netip.MustParseAddr("127.0.0.1"), srv, zap.NewNop(), nil)
	ss.srpIDHead = 1

	policy := rsvpPolicy()
	expectedMsg, err := pcep.NewPCInitiateMessageRSVP(1, policy.Name, false, 0,
		[]table.SegmentRSVPIPv4{
			table.NewSegmentRSVPIPv4(netip.MustParseAddr("10.26.10.17"), true),
			table.NewSegmentRSVPIPv4(netip.MustParseAddr("10.26.10.14"), true),
		}, policy.SrcAddr, policy.DstAddr)
	require.NoError(t, err)
	expected, err := expectedMsg.Serialize()
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() { errCh <- ss.SendPCInitiate(policy, false) }()

	got := make([]byte, len(expected))
	require.NoError(t, cli.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, err = io.ReadFull(cli, got)
	require.NoError(t, err)
	require.NoError(t, <-errCh)

	assert.Equal(t, expected, got, "session must send the RSVP-TE PCInitiate bytes")
	assert.Equal(t, uint32(2), ss.srpIDHead, "SRP-ID must increment after send")

	// Sanity: the message decodes as a PCInitiate (LSP-Init-Req).
	assert.Equal(t, uint8(pcep.MessageTypeLSPInitReq), got[1])
}

// SendPCUpdate with RSVP segments must likewise emit the RSVP PCUpd bytes.
func TestSessionSendPCUpdateRSVP(t *testing.T) {
	srv, cli := loopbackConns(t)
	defer srv.Close()
	defer cli.Close()

	ss := NewSession(0, netip.MustParseAddr("127.0.0.1"), srv, zap.NewNop(), nil)
	ss.srpIDHead = 5

	policy := rsvpPolicy()
	policy.PlspID = 42

	errCh := make(chan error, 1)
	go func() { errCh <- ss.SendPCUpdate(policy) }()

	header := make([]byte, 4)
	require.NoError(t, cli.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, err := io.ReadFull(cli, header)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, uint8(pcep.MessageTypeUpdate), header[1])
	assert.Equal(t, uint32(6), ss.srpIDHead)
}
