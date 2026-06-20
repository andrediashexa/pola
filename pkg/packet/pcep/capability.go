// Copyright (c) 2022 NTT Communications Corporation
//
// This software is released under the MIT License.
// see https://github.com/nttcom/pola/blob/main/LICENSE

package pcep

type CapabilityInterface interface {
	TLVInterface
	CapStrings() []string
}

func PolaCapability(caps []CapabilityInterface) []CapabilityInterface {
	polaCaps := []CapabilityInterface{}
	for _, cap := range caps {
		switch tlv := cap.(type) {
		case *StatefulPCECapability:
			tlv = &StatefulPCECapability{
				LSPUpdateCapability:            true,
				IncludeDBVersion:               false,
				LSPInstantiationCapability:     true,
				TriggeredResync:                false,
				DeltaLSPSyncCapability:         false,
				TriggeredInitialSync:           false,
				P2mpCapability:                 false,
				P2mpLSPUpdateCapability:        false,
				P2mpLSPInstantiationCapability: false,
				LSPSchedulingCapability:        false,
				PdLSPCapability:                false,
				ColorCapability:                true,
				PathRecomputationCapability:    false,
				StrictPathCapability:           false,
				Relax:                          false,
			}
			polaCaps = append(polaCaps, tlv)
		case *PathSetupTypeCapability:
			// Advertise RSVP-TE (PST 0) in addition to whatever the PCC offered
			// (e.g. SR-TE), so the PCC accepts PCE-initiated RSVP-TE LSPs. Without
			// it the Huawei VRP rejects RSVP-TE PCInitiate with Error-Type 2
			// (capability not supported). SR-TE/SRv6 PSTs are preserved.
			hasRSVP := false
			for _, p := range tlv.PathSetupTypes {
				if p == PathSetupTypeRSVPTE {
					hasRSVP = true
					break
				}
			}
			if !hasRSVP {
				tlv.PathSetupTypes = append(Psts{PathSetupTypeRSVPTE}, tlv.PathSetupTypes...)
			}
			polaCaps = append(polaCaps, tlv)
		case *LSPDBVersion:
			continue
		default:
			polaCaps = append(polaCaps, tlv)
		}
	}
	return polaCaps
}
