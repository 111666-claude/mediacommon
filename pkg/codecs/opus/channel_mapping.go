package opus

import "fmt"

// Channel is the semantic role of an output channel.
// Values 0..29 follow the channel names defined by RFC 7845, section 5.1.1.1.
type Channel uint8

const (
	// ChannelLeft is the left (L) channel.
	ChannelLeft Channel = 0
	// ChannelRight is the right (R) channel.
	ChannelRight Channel = 1
	// ChannelCenter is the center (C) channel.
	ChannelCenter Channel = 2
	// ChannelLFE is the low frequency effects (LFE) channel.
	ChannelLFE Channel = 3
	// ChannelRearLeft is the rear/side left (RL) channel.
	ChannelRearLeft Channel = 4
	// ChannelRearRight is the rear/side right (RR) channel.
	ChannelRearRight Channel = 5
	// ChannelRearCenter is the rear center (RC) channel.
	ChannelRearCenter Channel = 6
	// ChannelSideLeft is the side left (SL) channel.
	ChannelSideLeft Channel = 7
	// ChannelSideRight is the side right (SR) channel.
	ChannelSideRight Channel = 8
	// ChannelTopCenter is the top center (TC) channel.
	ChannelTopCenter Channel = 9
	// ChannelTopFrontLeft is the top front left (TFL) channel.
	ChannelTopFrontLeft Channel = 10
	// ChannelTopFrontCenter is the top front center (TFC) channel.
	ChannelTopFrontCenter Channel = 11
	// ChannelTopFrontRight is the top front right (TFR) channel.
	ChannelTopFrontRight Channel = 12
	// ChannelTopRearLeft is the top rear left (TRL) channel.
	ChannelTopRearLeft Channel = 13
	// ChannelTopRearCenter is the top rear center (TRC) channel.
	ChannelTopRearCenter Channel = 14
	// ChannelTopRearRight is the top rear right (TRR) channel.
	ChannelTopRearRight Channel = 15
	// ChannelTopSideLeft is the top side left (TSL) channel.
	ChannelTopSideLeft Channel = 16
	// ChannelTopSideRight is the top side right (TSR) channel.
	ChannelTopSideRight Channel = 17
	// ChannelBottomCenter is the bottom center (BC) channel.
	ChannelBottomCenter Channel = 18
	// ChannelBottomFrontLeft is the bottom front left (BFL) channel.
	ChannelBottomFrontLeft Channel = 19
	// ChannelBottomFrontCenter is the bottom front center (BFC) channel.
	ChannelBottomFrontCenter Channel = 20
	// ChannelBottomFrontRight is the bottom front right (BFR) channel.
	ChannelBottomFrontRight Channel = 21
	// ChannelBottomRearLeft is the bottom rear left (BRL) channel.
	ChannelBottomRearLeft Channel = 22
	// ChannelBottomRearCenter is the bottom rear center (BRC) channel.
	ChannelBottomRearCenter Channel = 23
	// ChannelBottomRearRight is the bottom rear right (BRR) channel.
	ChannelBottomRearRight Channel = 24
	// ChannelSilence marks an unused output channel that must be rendered as silence.
	ChannelSilence Channel = 255
)

// String implements fmt.Stringer.
func (c Channel) String() string {
	switch c {
	case ChannelSilence:
		return "silence"
	case ChannelLeft:
		return "L"
	case ChannelRight:
		return "R"
	case ChannelCenter:
		return "C"
	case ChannelLFE:
		return "LFE"
	case ChannelRearLeft:
		return "RL"
	case ChannelRearRight:
		return "RR"
	case ChannelRearCenter:
		return "RC"
	case ChannelSideLeft:
		return "SL"
	case ChannelSideRight:
		return "SR"
	case ChannelTopCenter:
		return "TC"
	case ChannelTopFrontLeft:
		return "TFL"
	case ChannelTopFrontCenter:
		return "TFC"
	case ChannelTopFrontRight:
		return "TFR"
	case ChannelTopRearLeft:
		return "TRL"
	case ChannelTopRearCenter:
		return "TRC"
	case ChannelTopRearRight:
		return "TRR"
	case ChannelTopSideLeft:
		return "TSL"
	case ChannelTopSideRight:
		return "TSR"
	case ChannelBottomCenter:
		return "BC"
	case ChannelBottomFrontLeft:
		return "BFL"
	case ChannelBottomFrontCenter:
		return "BFC"
	case ChannelBottomFrontRight:
		return "BFR"
	case ChannelBottomRearLeft:
		return "BRL"
	case ChannelBottomRearCenter:
		return "BRC"
	case ChannelBottomRearRight:
		return "BRR"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(c))
	}
}

// ChannelMapping is a parsed Opus channel mapping.
// It corresponds to the "Channel mapping" section of an Opus identification
// header (RFC 7845, section 5.1.1) and only supports channel mapping families
// 0 and 1.
type ChannelMapping struct {
	// Family is the channel mapping family (0 or 1).
	Family uint8

	// StreamCount is the number of streams (N), one per Opus encoder.
	StreamCount int

	// CoupledStreamCount is the number of coupled (stereo) streams (M).
	CoupledStreamCount int

	// Mapping is the per-output-channel mapping table.
	// A value of 255 means the output channel is silence.
	Mapping []uint8

	// Layout is the semantic role of each output channel, in output order.
	Layout []Channel
}

// family1Layouts contains the Vorbis-order output channel layouts defined by
// RFC 7845, section 5.1.1.2 for channel mapping family 1, indexed by channel
// count.
var family1Layouts = map[int][]Channel{
	1: {ChannelCenter},
	2: {ChannelLeft, ChannelRight},
	3: {ChannelLeft, ChannelCenter, ChannelRight},
	4: {ChannelLeft, ChannelRight, ChannelRearLeft, ChannelRearRight},
	5: {ChannelLeft, ChannelCenter, ChannelRight, ChannelRearLeft, ChannelRearRight},
	6: {
		ChannelLeft, ChannelCenter, ChannelRight,
		ChannelRearLeft, ChannelRearRight, ChannelLFE,
	},
	7: {
		ChannelLeft, ChannelCenter, ChannelRight,
		ChannelSideLeft, ChannelSideRight, ChannelRearCenter, ChannelLFE,
	},
	8: {
		ChannelLeft, ChannelCenter, ChannelRight,
		ChannelSideLeft, ChannelSideRight,
		ChannelRearLeft, ChannelRearRight, ChannelLFE,
	},
}

// Unmarshal decodes the channel mapping of an Opus identification header.
func (m *ChannelMapping) Unmarshal(h *IDHeader) error {
	channelCount := int(h.ChannelCount)

	switch h.ChannelMappingFamily {
	case 0:
		if len(h.ChannelMappingTable) != 0 {
			return fmt.Errorf("channel mapping table must be empty for family 0, got %d bytes",
				len(h.ChannelMappingTable))
		}

		if channelCount != 1 && channelCount != 2 {
			return fmt.Errorf("family 0 supports 1 or 2 channels, got %d", channelCount)
		}

		m.Family = 0
		m.CoupledStreamCount = channelCount - 1
		m.StreamCount = channelCount - m.CoupledStreamCount
		m.Mapping = nil
		m.Layout = family1Layouts[channelCount]

		return nil

	case 1:
		// stream count (1 byte) + coupled stream count (1 byte) + one byte
		// of mapping per output channel.
		if len(h.ChannelMappingTable) != 2+channelCount {
			return fmt.Errorf(
				"channel mapping table length is %d, expected %d for %d channels",
				len(h.ChannelMappingTable), 2+channelCount, channelCount)
		}

		m.StreamCount = int(h.ChannelMappingTable[0])
		m.CoupledStreamCount = int(h.ChannelMappingTable[1])
		m.Mapping = append([]uint8(nil), h.ChannelMappingTable[2:]...)

		if m.StreamCount == 0 {
			return fmt.Errorf("stream count must not be zero")
		}

		if m.CoupledStreamCount > m.StreamCount {
			return fmt.Errorf("coupled stream count (%d) exceeds stream count (%d)",
				m.CoupledStreamCount, m.StreamCount)
		}

		if m.StreamCount+m.CoupledStreamCount > 255 {
			return fmt.Errorf("stream count plus coupled stream count (%d) exceeds 255",
				m.StreamCount+m.CoupledStreamCount)
		}

		// RFC 7845, section 5.1.1: each mapping entry must either be 255
		// (silence) or smaller than M+N. Indices 0..2M-1 address the two
		// channels of a coupled stream, indices 2M..M+N-1 the uncoupled
		// (mono) streams. Reusing an index is permitted by the RFC, but
		// mediacommon rejects duplicates so that every output channel has
		// an unambiguous source.
		maxIndex := m.StreamCount + m.CoupledStreamCount
		seen := make(map[uint8]bool, channelCount)

		for i, idx := range m.Mapping {
			if idx == 255 {
				continue
			}

			if int(idx) >= maxIndex {
				return fmt.Errorf("channel mapping index %d is out of range (max %d)",
					idx, maxIndex-1)
			}

			if seen[idx] {
				return fmt.Errorf("duplicated channel mapping index %d at position %d",
					idx, i)
			}
			seen[idx] = true
		}

		layout, ok := family1Layouts[channelCount]
		if !ok {
			return fmt.Errorf("family 1 does not define a layout for %d channels", channelCount)
		}

		// Silence entries replace the role that the well-known layout assigns
		// to the corresponding output position.
		m.Layout = make([]Channel, channelCount)
		copy(m.Layout, layout)

		for i, idx := range m.Mapping {
			if idx == 255 {
				m.Layout[i] = ChannelSilence
			}
		}

		m.Family = 1

		return nil

	default:
		return fmt.Errorf("unsupported channel mapping family: %d", h.ChannelMappingFamily)
	}
}

// NewChannelMapping builds a channel mapping for family 1 and validates it.
// mapping is the per-output-channel mapping table; entries equal to 255 mark
// silent output channels.
func NewChannelMapping(streamCount, coupledStreamCount int, mapping []uint8) (*ChannelMapping, error) {
	m := &ChannelMapping{
		Family:             1,
		StreamCount:        streamCount,
		CoupledStreamCount: coupledStreamCount,
		Mapping:            append([]uint8(nil), mapping...),
	}

	if streamCount == 0 {
		return nil, fmt.Errorf("stream count must not be zero")
	}

	if coupledStreamCount > streamCount {
		return nil, fmt.Errorf("coupled stream count (%d) exceeds stream count (%d)",
			coupledStreamCount, streamCount)
	}

	if streamCount+coupledStreamCount > 255 {
		return nil, fmt.Errorf("stream count plus coupled stream count (%d) exceeds 255",
			streamCount+coupledStreamCount)
	}

	if len(mapping) == 0 || len(mapping) > 255 {
		return nil, fmt.Errorf("channel count must be between 1 and 255, got %d", len(mapping))
	}

	maxIndex := streamCount + coupledStreamCount
	// Reusing a mapping index is permitted by RFC 7845, but it is rejected
	// here so that every output channel has an unambiguous source.
	seen := make(map[uint8]bool, len(mapping))

	for i, idx := range mapping {
		if idx == 255 {
			continue
		}

		if int(idx) >= maxIndex {
			return nil, fmt.Errorf("channel mapping index %d is out of range (max %d)",
				idx, maxIndex-1)
		}

		if seen[idx] {
			return nil, fmt.Errorf("duplicated channel mapping index %d at position %d",
				idx, i)
		}
		seen[idx] = true
	}

	layout, ok := family1Layouts[len(mapping)]
	if !ok {
		return nil, fmt.Errorf("family 1 does not define a layout for %d channels", len(mapping))
	}

	m.Layout = make([]Channel, len(mapping))
	copy(m.Layout, layout)

	for i, idx := range mapping {
		if idx == 255 {
			m.Layout[i] = ChannelSilence
		}
	}

	return m, nil
}

// ToIDHeader fills the channel mapping family and table of an IDHeader.
// The remaining IDHeader fields (version, channel count, pre-skip, sample
// rate and output gain) are left untouched.
func (m ChannelMapping) ToIDHeader(h *IDHeader) {
	h.ChannelMappingFamily = m.Family

	switch m.Family {
	case 0:
		h.ChannelMappingTable = nil

	default:
		table := make([]byte, 0, 2+len(m.Mapping))
		table = append(table, byte(m.StreamCount), byte(m.CoupledStreamCount))
		table = append(table, m.Mapping...)
		h.ChannelMappingTable = table
	}
}
