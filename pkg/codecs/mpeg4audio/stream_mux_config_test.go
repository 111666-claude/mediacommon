package mpeg4audio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var streamMuxConfigCases = []struct {
	name string
	enc  []byte
	dec  StreamMuxConfig
}{
	{
		"lc",
		[]byte{0x40, 0x00, 0x26, 0x20, 0x3f, 0xc0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   24000,
						ChannelCount: 2,
					},
					LatmBufferFullness: 255,
				}},
			}},
		},
	},
	{
		"he-aac",
		[]byte{0x40, 0x00, 0x26, 0x10, 0x3f, 0xc0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   24000,
						ChannelCount: 1,
					},
					LatmBufferFullness: 255,
				}},
			}},
		},
	},
	{
		"sbr",
		[]byte{0x40, 0x00, 0x56, 0x23, 0x10, 0x1f, 0xe0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:                2,
						ExtensionType:       5,
						ExtensionSampleRate: 48000,
						SampleRate:          24000,
						ChannelCount:        2,
					},
					LatmBufferFullness: 255,
				}},
			}},
		},
	},
	{
		"ps",
		[]byte{0x40, 0x01, 0xd6, 0x13, 0x10, 0x1f, 0xe0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:                2,
						ExtensionType:       29,
						ExtensionSampleRate: 48000,
						SampleRate:          24000,
						ChannelCount:        1,
					},
					LatmBufferFullness: 255,
				}},
			}},
		},
	},
	{
		"other data and checksum",
		[]byte{0x40, 0x00, 0x24, 0x10, 0xad, 0xca, 0x00},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 1,
					},
					FrameLengthType: 2,
				}},
			}},
			OtherDataPresent: true,
			OtherDataLenBits: 220,
			CRCCheckPresent:  true,
			CRCCheckSum:      64,
		},
	},
	{
		"other data > 256 and checksum",
		[]byte{0x40, 0x00, 0x24, 0x10, 0xb0, 0x33, 0x85, 0x0},
		StreamMuxConfig{
			Programs: []*StreamMuxConfigProgram{{
				Layers: []*StreamMuxConfigLayer{{
					AudioSpecificConfig: &AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 1,
					},
					FrameLengthType: 2,
				}},
			}},
			OtherDataPresent: true,
			OtherDataLenBits: 880,
			CRCCheckPresent:  true,
			CRCCheckSum:      64,
		},
	},
}

func TestStreamMuxConfigUnmarshal(t *testing.T) {
	for _, ca := range streamMuxConfigCases {
		t.Run(ca.name, func(t *testing.T) {
			var dec StreamMuxConfig
			err := dec.Unmarshal(ca.enc)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)
		})
	}
}

func TestStreamMuxConfigUnmarshalTruncated(t *testing.T) {
	var dec StreamMuxConfig
	err := dec.Unmarshal([]byte{0x40, 0x00, 0x23, 0x10})
	require.NoError(t, err)
	require.Equal(t, StreamMuxConfig{
		Programs: []*StreamMuxConfigProgram{{
			Layers: []*StreamMuxConfigLayer{{
				AudioSpecificConfig: &AudioSpecificConfig{
					Type:         2,
					SampleRate:   48000,
					ChannelCount: 1,
				},
				LatmBufferFullness: 255,
				truncated:          true,
			}},
		}},
	}, dec)

	// the truncated config must still be marshalable without overrunning
	// the buffer or emitting frame length fields it never carried
	enc, err := dec.Marshal()
	require.NoError(t, err)
	require.Len(t, enc, dec.marshalSize())
}

func TestStreamMuxConfigMarshal(t *testing.T) {
	for _, ca := range streamMuxConfigCases {
		t.Run(ca.name, func(t *testing.T) {
			enc, err := ca.dec.Marshal()
			require.NoError(t, err)
			require.Equal(t, ca.enc, enc)
		})
	}
}

func TestStreamMuxConfigTruncatedMarshalNoOverrun(t *testing.T) {
	// Inputs whose program_config_element byteAlign, combined with a
	// truncated layer, used to make Marshal overrun its allocated buffer.
	for _, b := range [][]byte{
		[]byte("A0 \x0e00121\xff\xff000000000000000000\xff0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\xffaX770+2\xff\xff\xff"),
		[]byte("A0 \x00010A0000000000\x0f000000000000000C0A000000\x00\x0000\x00"),
	} {
		var conf StreamMuxConfig
		err := conf.Unmarshal(b)
		if err != nil {
			continue
		}

		// Marshal must not panic and the produced length must match the
		// computed size exactly.
		enc, err := conf.Marshal()
		require.NoError(t, err)
		require.Len(t, enc, conf.marshalSize())
	}
}

func FuzzStreamMuxConfigUnmarshal(f *testing.F) {
	for _, ca := range streamMuxConfigCases {
		f.Add(ca.enc)
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		var conf StreamMuxConfig
		err := conf.Unmarshal(b)
		if err != nil {
			return
		}

		_, err = conf.Marshal()
		require.NoError(t, err)
	})
}
