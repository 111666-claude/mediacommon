package fmp4

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4/seekablebuffer"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
)

func TestInitMPEG4AudioChannelConfigZeroRoundTrip(t *testing.T) {
	// 13-byte AudioSpecificConfig with channelConfiguration = 0
	// and a program_config_element describing a stereo pair.
	rawConfig := []byte{0x11, 0x80, 0x00, 0xc4, 0x00, 0x00, 0x40, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05}

	var config mpeg4audio.AudioSpecificConfig
	require.NoError(t, config.Unmarshal(rawConfig))
	require.Equal(t, 2, config.ChannelCount)
	require.NotNil(t, config.ProgramConfig)

	init := Init{
		Tracks: []*InitTrack{{
			ID:        1,
			TimeScale: 48000,
			Codec: &codecs.MPEG4Audio{
				Config: config,
			},
		}},
	}

	// marshal the initialization block
	var buf seekablebuffer.Buffer
	require.NoError(t, init.Marshal(&buf))

	// unmarshal it back
	var decoded Init
	require.NoError(t, decoded.Unmarshal(bytes.NewReader(buf.Bytes())))
	require.Len(t, decoded.Tracks, 1)

	audioTrack, ok := decoded.Tracks[0].Codec.(*codecs.MPEG4Audio)
	require.True(t, ok)

	// channel meaning is preserved from the PCE, not forced to a fixed layout
	require.Equal(t, 2, audioTrack.Config.ChannelCount)
	require.NotNil(t, audioTrack.Config.ProgramConfig)
	require.Equal(t, config.ProgramConfig, audioTrack.Config.ProgramConfig)

	// the configuration is written back byte by byte
	reEncoded, err := audioTrack.Config.Marshal()
	require.NoError(t, err)
	require.Equal(t, rawConfig, reEncoded)
}
