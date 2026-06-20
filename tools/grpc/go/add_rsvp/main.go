package main

import (
	"context"
	"log"
	"net/netip"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/nttcom/pola/api/pola/v1"
)

func main() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPCEServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ss := netip.MustParseAddr("2804:3200:1000:f::1")     // SOS PCC session
	src := netip.MustParseAddr("10.26.10.9")              // head-end LSR-id (IPv4)
	dst := netip.MustParseAddr("170.150.220.254")         // tail-end
	r, err := c.CreateSRPolicy(ctx, &pb.CreateSRPolicyRequest{
		SrPolicy: &pb.SRPolicy{
			PcepSessionAddr: ss.AsSlice(),
			SrcAddr:         src.AsSlice(),
			DstAddr:         dst.AsSlice(),
			Color:           1,
			PolicyName:      "TEST-RSVP-POLA",
			SegmentList: []*pb.Segment{
				{Sid: "rsvp:10.26.10.17"},
				{Sid: "rsvp:10.26.10.14"},
			},
		},
		Asn:         65000,
		SidValidate: true, // -> disablePathCompute -> explicit branch (parses rsvp:)
	})
	if err != nil {
		log.Fatalf("CreateSRPolicy error: %v", err)
	}
	log.Printf("OK: %#v", r)
}
