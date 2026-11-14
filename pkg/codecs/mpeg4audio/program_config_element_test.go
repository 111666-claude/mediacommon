package mpeg4audio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 13-byte AudioSpecificConfig: AAC-LC, 48 kHz, channelConfiguration = 0,
// with a program_config_element describing a stereo pair (one CPE) and
// five comment bytes.
var pceStereoConfig = AudioSpecificConfig{
	Type:         ObjectTypeAACLC,
	SampleRate:   48000,
	ChannelCount: 2,
	ProgramConfig: &ProgramConfigElement{
		ElementInstanceTag:   0,
		Profile:              0,
		SampleRateIndex:      3,
		NumFrontElements:     1,
		NumSideElements:      0,
		NumBackElements:      0,
		NumLFEElements:       0,
		NumAssocDataElements: 0,
		NumValidCCElements:   0,
		FrontElements: []ProgramConfigElementItem{
			{IsCPE: true, ElementTag: 0},
		},
		Comments: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
	},
}

var pceStereoBytes = []byte{0x11, 0x80, 0x00, 0xc4, 0x00, 0x00, 0x40, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05}

// 5.1 layout: three front elements (SCE + two CPE), one side CPE and one LFE.
var pceSurroundConfig = AudioSpecificConfig{
	Type:         ObjectTypeAACLC,
	SampleRate:   48000,
	ChannelCount: 8,
	ProgramConfig: &ProgramConfigElement{
		ElementInstanceTag:   0,
		Profile:              1,
		SampleRateIndex:      3,
		NumFrontElements:     3,
		NumSideElements:      1,
		NumBackElements:      0,
		NumLFEElements:       1,
		NumAssocDataElements: 0,
		NumValidCCElements:   0,
		MonoMixdownPresent:   true,
		MonoMixdownElement:   1,
		FrontElements: []ProgramConfigElementItem{
			{IsCPE: false, ElementTag: 0},
			{IsCPE: true, ElementTag: 1},
			{IsCPE: true, ElementTag: 2},
		},
		SideElements: []ProgramConfigElementItem{
			{IsCPE: true, ElementTag: 3},
		},
		LFEElements: []ProgramConfigElementItem{
			{IsCPE: false, ElementTag: 4},
		},
	},
}

func TestProgramConfigElementChannelCount(t *testing.T) {
	require.Equal(t, 2, pceStereoConfig.ProgramConfig.ChannelCount())
	require.Equal(t, 8, pceSurroundConfig.ProgramConfig.ChannelCount())
}

func TestAudioSpecificConfigPCEUnmarshal(t *testing.T) {
	for _, ca := range []struct {
		name string
		enc  []byte
		dec  AudioSpecificConfig
	}{
		{
			"channel configuration 0 with stereo PCE (13 bytes)",
			pceStereoBytes,
			pceStereoConfig,
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var dec AudioSpecificConfig
			err := dec.Unmarshal(ca.enc)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)
			// the channel count must come from the PCE, not from a fixed default
			require.NotZero(t, dec.ProgramConfig)
			require.Equal(t, dec.ProgramConfig.ChannelCount(), dec.ChannelCount)
		})
	}
}

func TestAudioSpecificConfigPCEMarshal(t *testing.T) {
	for _, ca := range []AudioSpecificConfig{pceStereoConfig, pceSurroundConfig} {
		enc, err := ca.Marshal()
		require.NoError(t, err)

		var dec AudioSpecificConfig
		require.NoError(t, dec.Unmarshal(enc))
		require.Equal(t, ca, dec)

		// byte-exact round trip
		enc2, err := dec.Marshal()
		require.NoError(t, err)
		require.Equal(t, enc, enc2)
	}

	// the known 13-byte configuration must be written back byte by byte
	enc, err := pceStereoConfig.Marshal()
	require.NoError(t, err)
	require.Len(t, enc, 13)
	require.Equal(t, pceStereoBytes, enc)
}

func TestAudioSpecificConfigChannelConfigurationValues(t *testing.T) {
	for _, ca := range []struct {
		name    string
		enc     []byte
		wantErr bool
	}{
		{
			"0x1180 with PCE is accepted",
			pceStereoBytes,
			false,
		},
		{
			"0x1190 ordinary stereo is accepted",
			[]byte{0x11, 0x90, 0x08, 0x10},
			false,
		},
		{
			"0x11f0 reserved value is rejected",
			[]byte{0x11, 0xf0},
			true,
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var dec AudioSpecificConfig
			err := dec.Unmarshal(ca.enc)
			if ca.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestADTSChannelConfigurationZero(t *testing.T) {
	// 7-byte ADTS header, profile=AAC-LC, 48 kHz, channelConfiguration = 0,
	// followed by a two-byte access unit.
	frame := []byte{0xFF, 0xF1, 0x4C, 0x00, 0x01, 0x3F, 0xFC, 0xAA, 0xBB}

	var pkts ADTSPackets
	require.NoError(t, pkts.Unmarshal(frame))
	require.Len(t, pkts, 1)
	require.Equal(t, 0, pkts[0].ChannelCount)
	require.Equal(t, 48000, pkts[0].SampleRate)
	require.Equal(t, ObjectTypeAACLC, pkts[0].Type)
	require.Equal(t, []byte{0xAA, 0xBB}, pkts[0].AU)

	// the zero channel configuration must survive a round trip
	enc, err := pkts.Marshal()
	require.NoError(t, err)
	require.Equal(t, frame, enc)
}

func TestADTS1000FramesChannelConfigurationZero(t *testing.T) {
	// build a stream of 1000 ADTS frames with channelConfiguration = 0
	frame := []byte{0xFF, 0xF1, 0x4C, 0x00, 0x01, 0x3F, 0xFC, 0xAA, 0xBB}
	stream := make([]byte, 0, 1000*len(frame))
	original := make([][]byte, 1000)
	for i := range 1000 {
		stream = append(stream, frame...)
		original[i] = []byte{0xAA, 0xBB}
	}

	var pkts ADTSPackets
	require.NoError(t, pkts.Unmarshal(stream))
	require.Len(t, pkts, 1000)

	for i, pkt := range pkts {
		require.Equal(t, 0, pkt.ChannelCount, "frame %d", i)
		require.Equal(t, original[i], pkt.AU, "frame %d", i)
	}

	// every frame is re-written byte by byte
	enc, err := pkts.Marshal()
	require.NoError(t, err)
	require.Equal(t, stream, enc)
}

func TestADTSInvalidChannelConfigurationStillRejected(t *testing.T) {
	// channelConfiguration is a 3-bit field, so 0..7 are encodable;
	// there is no reserved value reachable in ADTS, but a truncated
	// header must still surface an error boundary.
	frame := []byte{0xFF, 0xF1, 0x4C, 0x00, 0x01}
	var pkts ADTSPackets
	require.Error(t, pkts.Unmarshal(frame))
}
