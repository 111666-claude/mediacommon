package substructs

import (
	"fmt"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/opus"
)

// NewOpusAudioDescriptorFromIDHeader builds the opus_audio_descriptor that
// corresponds to the channel mapping of an Opus identification header.
// Only channel mapping families 0 and 1 are supported.
//
// Family 1 headers are carried in the explicit form (channel_config_code
// 0x81), which preserves the raw per-channel mapping. An error is returned if
// a mapping index cannot fit the descriptor's bit-packed field (for example a
// 255 silence marker with a small stream count).
func NewOpusAudioDescriptorFromIDHeader(h *opus.IDHeader) (*OpusAudioDescriptor, error) {
	m, err := h.ChannelMapping()
	if err != nil {
		return nil, err
	}

	return NewOpusAudioDescriptorFromChannelMapping(m)
}

// NewOpusAudioDescriptorFromChannelMapping builds the opus_audio_descriptor
// that corresponds to a parsed Opus channel mapping.
func NewOpusAudioDescriptorFromChannelMapping(m *opus.ChannelMapping) (*OpusAudioDescriptor, error) {
	channelCount := len(m.Layout)

	switch m.Family {
	case 0:
		// Family 0 is only defined for mono and stereo. Use the predefined
		// channel_config_code instead of the explicit form.
		switch channelCount {
		case 1:
			return &OpusAudioDescriptor{ChannelConfigCode: 0x01}, nil
		case 2:
			return &OpusAudioDescriptor{ChannelConfigCode: 0x02}, nil
		default:
			return nil, fmt.Errorf("family 0 supports 1 or 2 channels, got %d", channelCount)
		}

	case 1:
		desc := &OpusAudioDescriptor{
			ChannelConfigCode:    0x81,
			ExplicitChannelCount: channelCount,
			MappingFamily:        1,
			StreamCount:          m.StreamCount,
			CoupledStreamCount:   m.CoupledStreamCount,
			ChannelMapping:       append([]byte(nil), m.Mapping...),
		}

		// Make sure the mapping can be represented in the descriptor's
		// bit-packed form without truncating an index.
		if err := desc.checkExplicitMapping(); err != nil {
			return nil, err
		}

		return desc, nil

	default:
		return nil, fmt.Errorf("unsupported channel mapping family: %d", m.Family)
	}
}

// channelMappingFromExplicit converts the explicit (0x81) part of a
// descriptor into a parsed Opus channel mapping.
func (d OpusAudioDescriptor) channelMappingFromExplicit() (*opus.ChannelMapping, error) {
	if d.MappingFamily == 0 {
		m := &opus.ChannelMapping{Family: 0}
		switch d.ExplicitChannelCount {
		case 1:
			m.StreamCount = 1
			m.CoupledStreamCount = 0
			m.Layout = []opus.Channel{opus.ChannelCenter}
		case 2:
			m.StreamCount = 1
			m.CoupledStreamCount = 1
			m.Layout = []opus.Channel{opus.ChannelLeft, opus.ChannelRight}
		default:
			return nil, fmt.Errorf("unsupported channel_count for mapping_family 0: %d",
				d.ExplicitChannelCount)
		}
		return m, nil
	}

	// Reuse the opus package validation, including index bounds and silence
	// handling.
	m, err := opus.NewChannelMapping(d.StreamCount, d.CoupledStreamCount, d.ChannelMapping)
	if err != nil {
		return nil, err
	}

	if d.MappingFamily != 1 {
		return nil, fmt.Errorf("unsupported mapping family: %d", d.MappingFamily)
	}

	m.Family = 1

	return m, nil
}

// GetChannelMapping converts the descriptor into a parsed Opus channel
// mapping. Predefined channel_config_codes are expanded into the RFC 7845
// family 1 mapping they imply.
func (d *OpusAudioDescriptor) GetChannelMapping() (*opus.ChannelMapping, error) {
	if d.ChannelConfigCode == 0x81 {
		return d.channelMappingFromExplicit()
	}

	channelCount, ok := channelConfigTable[d.ChannelConfigCode]
	if !ok {
		return nil, fmt.Errorf("unsupported channel_config_code: 0x%02X", d.ChannelConfigCode)
	}

	switch d.ChannelConfigCode {
	case 0x01:
		return &opus.ChannelMapping{
			Family:             0,
			StreamCount:        1,
			CoupledStreamCount: 0,
			Layout:             []opus.Channel{opus.ChannelCenter},
		}, nil

	case 0x02:
		return &opus.ChannelMapping{
			Family:             0,
			StreamCount:        1,
			CoupledStreamCount: 1,
			Layout:             []opus.Channel{opus.ChannelLeft, opus.ChannelRight},
		}, nil

	case 0x82:
		// one channel, family 1 default: single uncoupled stream.
		return opus.NewChannelMapping(1, 0, []byte{0})

	case 0x83:
		// two channels, family 1 default: one coupled stream.
		return opus.NewChannelMapping(1, 1, []byte{0, 1})

	case 0x00, 0x80:
		// Dual mono: two independent mono signals, represented as the
		// uncoupled family 1 stereo mapping.
		return &opus.ChannelMapping{
			Family:             1,
			StreamCount:        2,
			CoupledStreamCount: 0,
			Mapping:            []byte{0, 1},
			Layout:             []opus.Channel{opus.ChannelLeft, opus.ChannelRight},
		}, nil

	default:
		// All other predefined codes (0x03..0x08 and their 0x8x aliases)
		// only carry a channel count. The mapping family and its default
		// table are implied by RFC 7845 family 1.
		pm, ok := predefinedMappings[channelCount]
		if !ok {
			return nil, fmt.Errorf("no predefined mapping for channel_config_code 0x%02X",
				d.ChannelConfigCode)
		}

		return opus.NewChannelMapping(pm.streamCount, pm.coupledStreamCount, pm.channelMapping)
	}
}

// IDHeader builds an Opus identification header from the descriptor.
// The fields not represented in the descriptor (pre-skip, input sample rate
// and output gain) are set to the values passed as arguments.
func (d *OpusAudioDescriptor) IDHeader(preSkip uint16, inputSampleRate uint32, outputGain uint16) (*opus.IDHeader, error) {
	m, err := d.GetChannelMapping()
	if err != nil {
		return nil, err
	}

	h := &opus.IDHeader{
		Version:         1,
		ChannelCount:    uint8(len(m.Layout)),
		PreSkip:         preSkip,
		InputSampleRate: inputSampleRate,
		OutputGain:      outputGain,
	}
	m.ToIDHeader(h)

	return h, nil
}
