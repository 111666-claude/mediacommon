package substructs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/opus"
)

// buildSilenceTable builds an OpusHead mapping table where the last of four
// output channels is silence. With three uncoupled streams the descriptor
// uses a 2-bit per-entry field, which cannot hold the silence marker 255, so
// this table is only used to validate the opus-side parsing.
func buildSilenceTable() []uint8 {
	return []uint8{3, 0, 0, 1, 2, 255}
}

func TestOpusDescriptorFromIDHeader(t *testing.T) {
	for _, ca := range []struct {
		name string
		h    *opus.IDHeader
		desc OpusAudioDescriptor
	}{
		{
			"mono family 0",
			&opus.IDHeader{
				Version:              1,
				ChannelCount:         1,
				ChannelMappingFamily: 0,
			},
			OpusAudioDescriptor{ChannelConfigCode: 0x01},
		},
		{
			"stereo family 0",
			&opus.IDHeader{
				Version:              1,
				ChannelCount:         2,
				ChannelMappingFamily: 0,
			},
			OpusAudioDescriptor{ChannelConfigCode: 0x02},
		},
		{
			"5.1 family 1",
			&opus.IDHeader{
				Version:              1,
				ChannelCount:         6,
				ChannelMappingFamily: 1,
				ChannelMappingTable:  []uint8{4, 2, 0, 4, 1, 2, 3, 5},
			},
			OpusAudioDescriptor{
				ChannelConfigCode:    0x81,
				ExplicitChannelCount: 6,
				MappingFamily:        1,
				StreamCount:          4,
				CoupledStreamCount:   2,
				ChannelMapping:       []byte{0, 4, 1, 2, 3, 5},
			},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			desc, err := NewOpusAudioDescriptorFromIDHeader(ca.h)
			require.NoError(t, err)
			require.Equal(t, &ca.desc, desc)

			// marshal/unmarshal must preserve the descriptor.
			enc, err := desc.Marshal()
			require.NoError(t, err)

			var decoded OpusAudioDescriptor
			err = decoded.Unmarshal(enc)
			require.NoError(t, err)
			require.Equal(t, desc, &decoded)

			// the descriptor must be able to rebuild the same channel mapping.
			m1, err := ca.h.ChannelMapping()
			require.NoError(t, err)

			m2, err := decoded.GetChannelMapping()
			require.NoError(t, err)
			require.Equal(t, m1, m2)
		})
	}
}

func TestOpusDescriptorSilence(t *testing.T) {
	// A mapping containing the silence marker 255 parses correctly on the
	// opus side.
	h := &opus.IDHeader{
		Version:              1,
		ChannelCount:         4,
		ChannelMappingFamily: 1,
		ChannelMappingTable:  buildSilenceTable(),
	}

	m, err := h.ChannelMapping()
	require.NoError(t, err)
	require.Equal(t, []opus.Channel{
		opus.ChannelLeft, opus.ChannelRight,
		opus.ChannelRearLeft, opus.ChannelSilence,
	}, m.Layout)

	// The descriptor's narrow bit-packed field cannot encode 255 here, so
	// converting it must fail with a clear error rather than truncate.
	_, err = NewOpusAudioDescriptorFromIDHeader(h)
	require.Error(t, err)
}

func TestOpusDescriptorRoundTrip(t *testing.T) {
	for _, ca := range []struct {
		name       string
		streamCnt  int
		coupledCnt int
		mapping    []uint8
	}{
		{"mono", 1, 0, []uint8{0}},
		{"stereo", 1, 1, []uint8{0, 1}},
		{"3ch", 2, 1, []uint8{0, 2, 1}},
		{"quad", 2, 2, []uint8{0, 1, 2, 3}},
		{"5ch", 3, 2, []uint8{0, 4, 1, 2, 3}},
		{"5.1", 4, 2, []uint8{0, 4, 1, 2, 3, 5}},
		{"6.1", 5, 2, []uint8{0, 4, 1, 2, 3, 5, 6}},
		{"7.1", 5, 3, []uint8{0, 6, 1, 4, 5, 2, 3, 7}},
	} {
		t.Run(ca.name, func(t *testing.T) {
			m, err := opus.NewChannelMapping(ca.streamCnt, ca.coupledCnt, ca.mapping)
			require.NoError(t, err)

			desc, err := NewOpusAudioDescriptorFromChannelMapping(m)
			require.NoError(t, err)

			enc, err := desc.Marshal()
			require.NoError(t, err)

			var decoded OpusAudioDescriptor
			err = decoded.Unmarshal(enc)
			require.NoError(t, err)

			roundTrip, err := decoded.GetChannelMapping()
			require.NoError(t, err)
			require.Equal(t, m, roundTrip)

			// and back into an Opus identification header.
			h, err := decoded.IDHeader(312, 48000, 0)
			require.NoError(t, err)
			require.Equal(t, uint8(len(ca.mapping)), h.ChannelCount)

			fromHeader, err := h.ChannelMapping()
			require.NoError(t, err)
			require.Equal(t, m, fromHeader)
		})
	}
}

func TestOpusDescriptorPredefinedCodes(t *testing.T) {
	for _, ca := range []struct {
		name       string
		code       uint8
		channelCnt int
		family     uint8
	}{
		{"0x01", 0x01, 1, 0},
		{"0x02", 0x02, 2, 0},
		{"0x03", 0x03, 3, 1},
		{"0x04", 0x04, 4, 1},
		{"0x05", 0x05, 5, 1},
		{"0x06", 0x06, 6, 1},
		{"0x07", 0x07, 7, 1},
		{"0x08", 0x08, 8, 1},
		{"0x82", 0x82, 1, 1},
		{"0x83", 0x83, 2, 1},
		{"0x84", 0x84, 3, 1},
		{"0x85", 0x85, 4, 1},
		{"0x86", 0x86, 5, 1},
		{"0x87", 0x87, 6, 1},
		{"0x88", 0x88, 7, 1},
	} {
		t.Run(ca.name, func(t *testing.T) {
			d := &OpusAudioDescriptor{ChannelConfigCode: ca.code}

			m, err := d.GetChannelMapping()
			require.NoError(t, err)
			require.Equal(t, ca.channelCnt, len(m.Layout))
			require.Equal(t, ca.family, m.Family)

			// every layout channel has a semantic role.
			for _, c := range m.Layout {
				require.NotEqual(t, opus.ChannelSilence, c)
			}
		})
	}
}

func TestOpusDescriptorDualMono(t *testing.T) {
	for _, code := range []uint8{0x00, 0x80} {
		d := &OpusAudioDescriptor{ChannelConfigCode: code}

		m, err := d.GetChannelMapping()
		require.NoError(t, err)
		require.Equal(t, 2, len(m.Layout))
		require.Equal(t, 2, m.StreamCount)
		require.Equal(t, 0, m.CoupledStreamCount)
	}
}

func TestOpusDescriptorConversionErrors(t *testing.T) {
	t.Run("unsupported family", func(t *testing.T) {
		h := &opus.IDHeader{
			Version:              1,
			ChannelCount:         2,
			ChannelMappingFamily: 2,
			ChannelMappingTable:  []uint8{1, 1, 0, 1},
		}
		_, err := NewOpusAudioDescriptorFromIDHeader(h)
		require.Error(t, err)
	})

	t.Run("unsupported mapping family in descriptor", func(t *testing.T) {
		d := &OpusAudioDescriptor{
			ChannelConfigCode:    0x81,
			ExplicitChannelCount: 3,
			MappingFamily:        255,
			StreamCount:          3,
			CoupledStreamCount:   0,
			ChannelMapping:       []byte{0, 1, 2},
		}
		_, err := d.GetChannelMapping()
		require.EqualError(t, err, "unsupported mapping family: 255")
	})

	t.Run("index does not fit bit-packed descriptor", func(t *testing.T) {
		// with 8 uncoupled streams the field is 4 bits wide, so the silence
		// marker 255 cannot be encoded.
		m, err := opus.NewChannelMapping(
			8, 0, []uint8{0, 255, 1, 2, 3, 4, 5, 6})
		require.NoError(t, err)

		_, err = NewOpusAudioDescriptorFromChannelMapping(m)
		require.Error(t, err)
	})

	t.Run("explicit check rejects inconsistent descriptor", func(t *testing.T) {
		d := &OpusAudioDescriptor{
			ChannelConfigCode:    0x81,
			ExplicitChannelCount: 2,
			MappingFamily:        1,
			StreamCount:          1,
			CoupledStreamCount:   2, // M > N
			ChannelMapping:       []byte{0, 1},
		}
		require.Error(t, d.checkExplicitMapping())

		// the wire codec itself stays lossless and never fails.
		enc, err := d.Marshal()
		require.NoError(t, err)
		require.NotEmpty(t, enc)
	})
}

func TestOpusDescriptorIDHeaderFields(t *testing.T) {
	d := &OpusAudioDescriptor{ChannelConfigCode: 0x06}

	h, err := d.IDHeader(312, 48000, 100)
	require.NoError(t, err)
	require.Equal(t, uint8(1), h.Version)
	require.Equal(t, uint8(6), h.ChannelCount)
	require.Equal(t, uint16(312), h.PreSkip)
	require.Equal(t, uint32(48000), h.InputSampleRate)
	require.Equal(t, uint16(100), h.OutputGain)
	require.Equal(t, uint8(1), h.ChannelMappingFamily)
}
